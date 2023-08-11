package apis

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"strings"

	"git.yunify.com/quanxiang/workflow/internal/common"

	"github.com/go-kit/log"
	"github.com/quanxiang-cloud/cabin/tailormade/resp"

	triggerapis "git.yunify.com/quanxiang/trigger/apis"
	triggerclient "git.yunify.com/quanxiang/trigger/pkg/client"
	"git.yunify.com/quanxiang/workflow/apis"
	"git.yunify.com/quanxiang/workflow/pkg/apis/v1alpha1"
	"git.yunify.com/quanxiang/workflow/pkg/client/clientset/versioned"
	"git.yunify.com/quanxiang/workflow/pkg/mid/service"
	"git.yunify.com/quanxiang/workflow/pkg/node"
	quanxiangform "git.yunify.com/quanxiang/workflow/pkg/node/nodes/quanxiang_form"
	"github.com/go-kit/kit/endpoint"
)

type Endpoints struct {
	node.Endpoints
	// portal
	SaveFlowEndpoint         endpoint.Endpoint
	UpdateFlowStatusEndpoint endpoint.Endpoint
	DeleteFlowEndpoint       endpoint.Endpoint
	GetFlowInfoEndpoint      endpoint.Endpoint
	GetFlowListEndpoint      endpoint.Endpoint

	SaveFlowVariableEndpoint    endpoint.Endpoint
	DeleteFlowVariableEndpoint  endpoint.Endpoint
	GetFlowVariableListEndpoint endpoint.Endpoint

	// home
	RecallEndpoint                 endpoint.Endpoint //cancel
	UrgeEndpoint                   endpoint.Endpoint
	ExamineEndpoint                endpoint.Endpoint //审核，reviewTask
	GetFormDataEndpoint            endpoint.Endpoint //获取表单数据，getFormData
	GetFlowProcessEndpoint         endpoint.Endpoint //获取流程审核进度，processHistories
	GetFormFieldPermissionEndpoint endpoint.Endpoint //获取表单字段权限，getFlowInstanceForm
	CCToMeListEndpoint             endpoint.Endpoint //抄送给我的列表
	ExaminedListEndpoint           endpoint.Endpoint //已审核的列表，reviewedList
	PendingExamineListEndpoint     endpoint.Endpoint //待审核的，waitReviewList
	PendingExamineCountEndpoint    endpoint.Endpoint //待审核的条数，getFlowInstanceCount
	MyApplyListEndpoint            endpoint.Endpoint //我发起的，myApplyList

	AppReplicationExportEndpoint endpoint.Endpoint
	AppReplicationImportEndpoint endpoint.Endpoint
}

func NewEndPoints(logger log.Logger, wl versioned.Client, trigger triggerclient.Trigger, confMysql common.Mysql, workFlowInstance, homeHost string) Endpoints {
	end := Endpoints{}
	s := service.NewOldFlow(logger, confMysql, workFlowInstance, homeHost)
	end.SaveFlowEndpoint = SaveFlowEndpoint(s, wl)
	end.UpdateFlowStatusEndpoint = UpdateFlowStatusEndpoint(s, trigger)
	end.DeleteFlowEndpoint = DeleteFlowEndpoint(s)
	end.GetFlowInfoEndpoint = GetFlowInfoEndpoint(s)
	end.GetFlowListEndpoint = GetFlowListEndpoint(s)

	end.SaveFlowVariableEndpoint = SaveFlowVariableEndpoint(s, wl)
	end.DeleteFlowVariableEndpoint = DeleteFlowVariableEndpoint(s)
	end.GetFlowVariableListEndpoint = GetFlowVariableListEndpoint(s)

	end.GetFormDataEndpoint = GetFormDataEndpoint(s)
	end.GetFlowProcessEndpoint = GetFlowProcessEndpoint(s)
	end.GetFormFieldPermissionEndpoint = GetFormFieldPermissionEndpoint(s)
	end.CCToMeListEndpoint = CCToMeListEndpoint(s)
	end.ExaminedListEndpoint = ExaminedListEndpoint(s)
	end.PendingExamineListEndpoint = PendingExamineListEndpoint(s)
	end.PendingExamineCountEndpoint = PendingExamineCountEndpoint(s)
	end.MyApplyListEndpoint = MyApplyListEndpoint(s)

	end.ExamineEndpoint = ExamineEndpoint(s)
	end.UrgeEndpoint = UrgeEndpoint(s)
	end.RecallEndpoint = RecallEndpoint(s)
	end.AppReplicationExportEndpoint = AppReplicationExportEndpoint(s, wl)
	end.AppReplicationImportEndpoint = AppReplicationImportEndpoint(s, wl)
	return end
}
func SaveFlowEndpoint(s service.OldFlow, wl versioned.Client) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.SaveFlowRequest)

		p := &ProcessModel{}
		err := json.Unmarshal([]byte(req.BpmnText), p)
		if err != nil {
			return nil, err
		}

		node := getFirstNode(p)

		if req.TriggerMode == "FORM_DATA" {
			req.FormID = node.Data.BusinessData["form"].(map[string]interface{})["value"].(string)
		}
		response, err := s.SaveFlow(ctx, req)
		if err != nil {
			return response, err
		}
		err = savePipline(ctx, response.ID, req.AppID, req.FormID, req.BpmnText, req.TriggerMode, req.UserID, s, wl)
		if err != nil {
			return nil, err
		}
		r := resp.Format(response, err)
		return r, err
	}
}

// NodeDataModel info
type NodeDataModel struct {
	Name                  string   `json:"name"`
	BranchTargetElementID string   `json:"branchTargetElementID"`
	ChildrenID            []string `json:"childrenID"`
	BranchID              string   `json:"branchID"`
}

// ShapeDataModel info
type ShapeDataModel struct {
	NodeData     NodeDataModel          `json:"nodeData"`
	BusinessData map[string]interface{} `json:"businessData"`
}

// ShapeModel struct
type ShapeModel struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Data   ShapeDataModel `json:"data"`
	Source string         `json:"source"`
	Target string         `json:"target"`
}

// ProcessModel struct
type ProcessModel struct {
	Version string       `json:"version"`
	Shapes  []ShapeModel `json:"shapes"`
}

func getFirstNode(p *ProcessModel) *ShapeModel {
	for _, elem := range p.Shapes {
		if elem.Type == "formData" {
			return &elem
		}
	}
	return nil
}

func getNode(p *ProcessModel, id string) *ShapeModel {
	for _, elem := range p.Shapes {
		if elem.ID == id {
			return &elem
		}
	}
	return nil
}
func savePipline(ctx context.Context, flowID, appID, formID, flowBpm, triggerModel, flowCreatedBy string, s service.OldFlow, wl versioned.Client) error {
	pipeline := &v1alpha1.Pipeline{}
	//TODO: 流程变量

	// 流程转换为 pipeline

	p := &ProcessModel{}
	err := json.Unmarshal([]byte(flowBpm), p)
	if err != nil {
		return err
	}

	node := getFirstNode(p)

	pipeline.Name = flowID
	pipeline.Spec.Params = []v1alpha1.ParamSpec{
		{
			Name:        "appID",
			Default:     appID,
			Description: "lowcode application id",
		},
		{
			Name:        "tableID",
			Default:     node.Data.BusinessData["form"].(map[string]interface{})["value"].(string),
			Description: "lowcode table id",
		}, {
			Name:        "dataID",
			Description: "lowcode form data id",
		},
	}
	// FIXME:
	flowVars, err := s.GetFlowVariableListByFlowID(ctx, &service.GetFlowVariableListByFlowIDRequest{
		FlowID: flowID,
	})
	if err != nil {
		return err
	}
	for k := range flowVars.List {
		pipeline.Spec.Communal = append(pipeline.Spec.Communal, v1alpha1.ParamSpec{
			Name:        flowVars.List[k].Code,
			Description: flowVars.List[k].Desc,
			Default:     flowVars.List[k].DefaultValue,
		})
	}

	nodeID := make(map[string]struct{})

	nodes := []*ShapeModel{getNode(p, node.Data.NodeData.ChildrenID[0])}
DONE:
	for len(nodes) != 0 {
		cp := nodes
		nodes = make([]*ShapeModel, 0)
		for index, node := range cp {
			if _, ok := nodeID[node.ID]; ok {
				continue
			}
			nodeID[node.ID] = struct{}{}

			pn := v1alpha1.Node{
				Name: node.ID,
			}

			switch node.Type {
			case "end":
				if len(nodes) == 1 {
					break DONE
				}
				continue
			case "email":
				pn.Spec.Type = "email"
				pn.Spec.Params = []*v1alpha1.KeyAndValue{
					{
						Key:   "appID",
						Value: "$(params.appID)",
					},
					{
						Key:   "tableID",
						Value: "$(params.tableID)",
					},
					{
						Key:   "dataID",
						Value: "$(params.dataID)",
					},
				}

				if title, ok := node.Data.BusinessData["title"]; ok {
					pn.Spec.Params = append(pn.Spec.Params, &v1alpha1.KeyAndValue{
						Key:   "title",
						Value: title.(string),
					})
				}

				if content, ok := node.Data.BusinessData["content"]; ok {
					pn.Spec.Params = append(pn.Spec.Params, &v1alpha1.KeyAndValue{
						Key:   "content",
						Value: content.(string),
					})
				}
				switch node.Data.BusinessData["approvePersons"].(map[string]interface{})["type"].(string) {
				case "person":
					var users []string
					for _, user := range node.Data.BusinessData["approvePersons"].(map[string]interface{})["users"].([]interface{}) {
						users = append(users, fmt.Sprintf("email.%s", user.(map[string]interface{})["email"].(string)))
					}

					pn.Spec.Params = append(pn.Spec.Params, &v1alpha1.KeyAndValue{
						Key:   "to",
						Value: strings.Join(users, ","),
					})
				case "field":
					for _, fields := range node.Data.BusinessData["approvePersons"].(map[string]interface{})["fields"].([]interface{}) {
						if fields, ok := fields.(string); ok {
							pn.Spec.Params = append(pn.Spec.Params, &v1alpha1.KeyAndValue{
								Key:   "to",
								Value: fmt.Sprintf("fields.%s", fields),
							})
						}
					}
				case "superior":
					pn.Spec.Params = append(pn.Spec.Params, &v1alpha1.KeyAndValue{
						Key:   "to",
						Value: "superior",
					})
				case "processInitiator":
					pn.Spec.Params = append(pn.Spec.Params, &v1alpha1.KeyAndValue{
						Key:   "to",
						Value: "processInitiator",
					})
				}
			case "approve":
				pn.Spec.Type = "approve"
				pn.Spec.OutPut = []string{"agree"}
				pn.Spec.Params = []*v1alpha1.KeyAndValue{
					{
						Key:   "appID",
						Value: "$(params.appID)",
					},
					{
						Key:   "tableID",
						Value: "$(params.tableID)",
					},
					{
						Key:   "dataID",
						Value: "$(params.dataID)",
					},
					{
						Key:   "flowID",
						Value: flowID,
					},
					{
						Key:   "nodeDefKey",
						Value: node.ID,
					},
					{
						Key:   "flowCreatedBy",
						Value: flowCreatedBy,
					},
				}
				for k := range flowVars.List {
					if flowVars.List[k].Name == "SYS_AUDIT_BOOL" {
						pn.Spec.Params = append(pn.Spec.Params, &v1alpha1.KeyAndValue{
							Key:   "SYS_AUDIT_BOOL",
							Value: flowVars.List[k].Code,
						})

						pn.Spec.Params = append(pn.Spec.Params, &v1alpha1.KeyAndValue{
							Key:   flowVars.List[k].Code,
							Value: fmt.Sprintf("$(communal.%s)", flowVars.List[k].Code),
						})
						break
					}
				}

				dealUsers := make([]string, 0)
				if basicConfig, ok := node.Data.BusinessData["basicConfig"].(map[string]interface{}); basicConfig != nil && ok {
					pn.Spec.Params = append(pn.Spec.Params, &v1alpha1.KeyAndValue{
						Key:   "taskType",
						Value: basicConfig["multiplePersonWay"].(string),
					})

					if approvePersons, ok1 := basicConfig["approvePersons"].(map[string]interface{}); approvePersons != nil && ok1 {
						switch approvePersons["type"] {
						case "person":
							if users, ok2 := approvePersons["users"].([]interface{}); users != nil && ok2 {
								for k := range users {
									if u, ok3 := users[k].(map[string]interface{}); u != nil && ok3 {
										dealUsers = append(dealUsers, "person."+u["id"].(string))
									}
								}
							}
						case "field":
							if fields, ok2 := approvePersons["fields"].([]interface{}); fields != nil && ok2 {
								for k := range fields {
									dealUsers = append(dealUsers, "field."+fields[k].(string))
								}
							}
						case "superior":
							dealUsers = append(dealUsers, "leader")
						case "processInitiator":
							dealUsers = append(dealUsers, "formApplyUserID")
						}
					}
				}
				pn.Spec.Params = append(pn.Spec.Params, &v1alpha1.KeyAndValue{
					Key:   "dealUsers",
					Value: strings.Join(dealUsers, ","),
				})

			case "processBranch":
				pn.Spec.Type = "process-branch"
				pn.Spec.Params = []*v1alpha1.KeyAndValue{
					{
						Key:   "appID",
						Value: "$(params.appID)",
					},
					{
						Key:   "tableID",
						Value: "$(params.tableID)",
					},
					{
						Key:   "dataID",
						Value: "$(params.dataID)",
					},
				}
				if rule, ok := node.Data.BusinessData["rule"]; ok {
					if rule, ok := rule.(string); ok {
						pn.Spec.Params = append(pn.Spec.Params, &v1alpha1.KeyAndValue{
							Key:   "rule",
							Value: rule,
						})

						for _, expr := range quanxiangform.ParseExpr(rule) {
							if strings.HasPrefix(expr.Key, "$flowVar_") {
								pn.Spec.Params = append(pn.Spec.Params, &v1alpha1.KeyAndValue{
									Key:   expr.Key[1:],
									Value: fmt.Sprintf("$(communal.%s)", expr.Key[1:]),
								})
							}
						}

					}
				} else if index > 0 {
					pn.Spec.Params = append(pn.Spec.Params, &v1alpha1.KeyAndValue{
						Key:   "rule",
						Value: "[else] == false",
					})
					pn.Spec.Params = append(pn.Spec.Params, &v1alpha1.KeyAndValue{
						Key:   "else",
						Value: fmt.Sprintf("$(task.%s.output.ok)", cp[index-1].ID),
					})
				}
			case "tableDataCreate":
				pn.Spec.Type = "form-create-data"
				pn.Spec.Params = []*v1alpha1.KeyAndValue{
					{
						Key:   "appID",
						Value: "$(params.appID)",
					},
					{
						Key:   "tableID",
						Value: "$(params.tableID)",
					},
					{
						Key:   "dataID",
						Value: "$(params.dataID)",
					},
					{
						Key:   "targetTableID",
						Value: node.Data.BusinessData["targetTableId"].(string),
					},
				}

				if rule, ok := node.Data.BusinessData["createRule"].(map[string]interface{}); ok {
					vByte, _ := json.Marshal(rule)
					pn.Spec.Params = append(pn.Spec.Params, &v1alpha1.KeyAndValue{
						Key:   "createRule",
						Value: string(vByte),
					})
				}
			case "tableDataUpdate":
				pn.Spec.Type = "form-update-data"
				pn.Spec.Params = []*v1alpha1.KeyAndValue{
					{
						Key:   "appID",
						Value: "$(params.appID)",
					},
					{
						Key:   "tableID",
						Value: "$(params.tableID)",
					},
					{
						Key:   "dataID",
						Value: "$(params.dataID)",
					},
					{
						Key:   "targetTableID",
						Value: node.Data.BusinessData["targetTableId"].(string),
					},
				}

				if rule, ok := node.Data.BusinessData["updateRule"]; ok {
					vByte, _ := json.Marshal(rule)
					pn.Spec.Params = append(pn.Spec.Params, &v1alpha1.KeyAndValue{
						Key:   "updateRule",
						Value: string(vByte),
					})
				}

				if fileterRule, ok := node.Data.BusinessData["filterRule"]; ok {
					frByte, err := json.Marshal(fileterRule)
					if err == nil {
						pn.Spec.Params = append(pn.Spec.Params, &v1alpha1.KeyAndValue{
							Key:   "filterRule",
							Value: string(frByte),
						})
					}
				}
			case "webhook":
				pn.Spec.Type = "web-hook"
				pn.Spec.Params = []*v1alpha1.KeyAndValue{
					{
						Key:   "appID",
						Value: "$(params.appID)",
					},
					{
						Key:   "tableID",
						Value: "$(params.tableID)",
					},
					{
						Key:   "dataID",
						Value: "$(params.dataID)",
					},
				}

				if config, ok := node.Data.BusinessData["config"]; ok {
					cb, err := json.Marshal(config)
					if err == nil {
						pn.Spec.Params = append(pn.Spec.Params, &v1alpha1.KeyAndValue{
							Key:   "config",
							Value: string(cb),
						})
					}

				}
			}
			if node.Data.NodeData.BranchID != "" {
				pn.Spec.When = []v1alpha1.When{
					{
						Input:    fmt.Sprintf("$(task.%s.output.ok)", node.Data.NodeData.BranchID),
						Operator: "eq",
						Values:   []string{"true"},
					},
				}
			}

			if pn.Spec.Type != "" {
				pipeline.Spec.Nodes = append(pipeline.Spec.Nodes, pn)
			}

			for _, childID := range node.Data.NodeData.ChildrenID {
				nodes = append(nodes, getNode(p, childID))
			}
		}

	}

	spl := &apis.SavePipeline{}
	spl.Pipeline = *pipeline
	err = wl.Save(ctx, spl)
	return err
}
func UpdateFlowStatusEndpoint(s service.OldFlow, trigger triggerclient.Trigger) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.UpdateFlowRequest)
		response, err := s.UpdateFlow(ctx, req)
		if err != nil {
			return nil, err
		}
		if response == nil {
			return nil, nil
		}
		// NodeDataModel info
		type NodeDataModel struct {
			Name                  string   `json:"name"`
			BranchTargetElementID string   `json:"branchTargetElementID"`
			ChildrenID            []string `json:"childrenID"`
		}

		// ShapeDataModel info
		type ShapeDataModel struct {
			NodeData     NodeDataModel          `json:"nodeData"`
			BusinessData map[string]interface{} `json:"businessData"`
		}

		// ShapeModel struct
		type ShapeModel struct {
			ID     string         `json:"id"`
			Type   string         `json:"type"`
			Data   ShapeDataModel `json:"data"`
			Source string         `json:"source"`
			Target string         `json:"target"`
		}

		// ProcessModel struct
		type ProcessModel struct {
			Version string       `json:"version"`
			Shapes  []ShapeModel `json:"shapes"`
		}

		p := &ProcessModel{}
		err = json.Unmarshal([]byte(response.FlowInfo.BpmnText), p)
		if err != nil {
			return response, err
		}
		var getFirstNode = func(p *ProcessModel) *ShapeModel {
			for _, elem := range p.Shapes {
				if elem.Type == "formData" {
					return &elem
				}
			}
			return nil
		}
		firstNode := getFirstNode(p)
		//todo 调用trigger create
		switch req.Status {
		case "DISABLE":
			err = trigger.Delete(ctx, &triggerapis.DeleteTrigger{
				Name: response.FlowInfo.ID,
			})
		case "ENABLE":
			triReq := &triggerapis.CreateTrigger{
				Name:         response.FlowInfo.ID,
				PipelineName: response.FlowInfo.ID,
			}
			if firstNode.Data.NodeData.Name == "工作表触发" {
				triReq.Type = "form"
			}
			var triggerDataType = ""
			triggerWays := firstNode.Data.BusinessData["triggerWay"].([]interface{})
			for k := range triggerWays {
				if triggerWays[k].(string) == "whenAdd" {
					triggerDataType = "CREATE"
				} else {
					triggerDataType = "UPDATE"
				}
				triReq.Data = map[string]interface{}{
					"app_id":   response.AppID,
					"table_id": response.FormID,
					"type":     triggerDataType,
				}
				err = trigger.Create(ctx, triReq)
			}

		}
		if err != nil {
			return nil, err
		}
		r := resp.Format(response.Flag, err)
		return r, nil
	}
}
func DeleteFlowEndpoint(s service.OldFlow) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.DeleteFlowRequest)
		response, err := s.DeleteFlow(ctx, req)
		r := resp.Format(response, err)
		return r, nil
	}
}
func GetFlowInfoEndpoint(s service.OldFlow) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.GetFlowInfoRequest)
		response, err := s.GetFlowInfo(ctx, req)
		r := resp.Format(response, err)
		return r, nil
	}
}
func GetFlowListEndpoint(s service.OldFlow) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.GetFlowListRequest)
		response, err := s.GetFlowList(ctx, req)
		r := resp.Format(response, err)
		return r, nil
	}
}
func SaveFlowVariableEndpoint(s service.OldFlow, wl versioned.Client) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.SaveFlowVariableRequest)
		response, err := s.SaveFlowVariable(ctx, req)
		flowInfo, err := s.GetFlowInfo(ctx, &service.GetFlowInfoRequest{
			ID: response.FlowID,
		})
		if err != nil {
			return nil, err
		}
		if flowInfo != nil {
			err = savePipline(ctx, flowInfo.ID, flowInfo.AppID, flowInfo.FormID, flowInfo.BpmnText, flowInfo.TriggerMode, flowInfo.CreatorID, s, wl)
			if err != nil {
				return nil, err
			}
		}

		r := resp.Format(response, err)
		return r, nil
	}
}
func DeleteFlowVariableEndpoint(s service.OldFlow) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.DeleteFlowVariableRequest)
		response, err := s.DeleteFlowVariable(ctx, req)
		r := resp.Format(response, err)
		return r, nil
	}
}

func GetFlowVariableListEndpoint(s service.OldFlow) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.GetFlowVariableListRequest)
		response, err := s.GetFlowVariableList(ctx, req)
		r := resp.Format(response.Data, err)
		return r, nil
	}
}

func GetFormDataEndpoint(s service.OldFlow) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.GetFormDataRequest)
		response, err := s.GetFormData(ctx, req)
		r := resp.Format(response.Data, err)
		return r, err
	}
}

func GetFlowProcessEndpoint(s service.OldFlow) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.GetFlowProcessRequest)
		response, err := s.GetFlowProcess(ctx, req)
		r := resp.Format(response.Data, err)
		return r, err
	}
}

func GetFormFieldPermissionEndpoint(s service.OldFlow) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.GetFormFieldPermissionRequest)
		response, err := s.GetFormFieldPermission(ctx, req)
		r := resp.Format(response, err)
		return r, err
	}
}

func CCToMeListEndpoint(s service.OldFlow) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.CCToMeListRequest)
		response, err := s.CCToMeList(ctx, req)
		r := resp.Format(response, err)
		return r, err
	}
}

func ExaminedListEndpoint(s service.OldFlow) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.ExaminedListRequest)
		response, err := s.ExaminedList(ctx, req)
		r := resp.Format(response, err)
		return r, err
	}
}

func PendingExamineListEndpoint(s service.OldFlow) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.PendingExamineListRequest)
		response, err := s.PendingExamineList(ctx, req)
		r := resp.Format(response, err)
		return r, err
	}
}

func PendingExamineCountEndpoint(s service.OldFlow) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.PendingExamineCountRequest)
		response, err := s.PendingExamineCount(ctx, req)
		r := resp.Format(response, err)
		return r, err
	}
}

func MyApplyListEndpoint(s service.OldFlow) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.MyApplyListRequest)
		response, err := s.MyApplyList(ctx, req)
		r := resp.Format(response, err)
		return r, err
	}
}

func ExamineEndpoint(s service.OldFlow) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.ExamineRequest)
		response, err := s.Examine(ctx, req)
		r := resp.Format(response, err)
		return r, err
	}
}

func UrgeEndpoint(s service.OldFlow) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.UrgeRequest)
		response, err := s.Urge(ctx, req)
		r := resp.Format(response, err)
		return r, err
	}
}
func RecallEndpoint(s service.OldFlow) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.RecallRequest)
		response, err := s.Recall(ctx, req)
		r := resp.Format(response, err)
		return r, err
	}
}
func AppReplicationExportEndpoint(s service.OldFlow, wl versioned.Client) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.AppExportRequest)
		response, err := s.AppExport(ctx, req)
		r := resp.Format(response.Data, err)
		return r, err
	}
}
func AppReplicationImportEndpoint(s service.OldFlow, wl versioned.Client) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.AppImportRequest)
		response, err := s.AppImport(ctx, req)
		if err == nil {
			for k := range response.ImportFlows {
				err = savePipline(ctx, response.ImportFlows[k].ID, response.ImportFlows[k].AppID, response.ImportFlows[k].FormID, response.ImportFlows[k].BpmnText, response.ImportFlows[k].TriggerMode, req.UserID, s, wl)
				if err != nil {
					return resp.Format(false, err), errors.Wrap(err, "import flow err")
				}
			}
		}
		r := resp.Format(response.Data, err)
		return r, err
	}
}

type ur interface {
	GetErr() error
	GetData() interface{}
}

type universalResponse struct {
	Err  error       `json:"err,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

func (u universalResponse) GetErr() error {
	return u.Err
}

func (u universalResponse) GetData() interface{} {
	return u.Data
}
