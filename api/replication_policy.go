/*
   Copyright (c) 2016 VMware, Inc. All Rights Reserved.
   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package api

import (
	"fmt"

	"net/http"
	"strconv"
	"time"

	"github.com/vmware/harbor/dao"
	"github.com/vmware/harbor/models"
	"github.com/vmware/harbor/utils/log"
)

// RepPolicyAPI handles /api/replicationPolicies /api/replicationPolicies/:id/enablement
type RepPolicyAPI struct {
	BaseAPI
}

// Prepare validates whether the user has system admin role
func (pa *RepPolicyAPI) Prepare() {
	uid := pa.ValidateUser()
	var err error
	isAdmin, err := dao.IsAdminRole(uid)
	if err != nil {
		log.Errorf("Failed to Check if the user is admin, error: %v, uid: %d", err, uid)
	}
	if !isAdmin {
		pa.CustomAbort(http.StatusForbidden, "")
	}
}

// Get ...
func (pa *RepPolicyAPI) Get() {
	id := pa.GetIDFromURL()
	policy, err := dao.GetRepPolicy(id)
	if err != nil {
		log.Errorf("failed to get policy %d: %v", id, err)
		pa.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	if policy == nil {
		pa.CustomAbort(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	}

	pa.Data["json"] = policy
	pa.ServeJSON()
}

// List filters policies by name and project_id, if name and project_id
// are nil, List returns all policies
func (pa *RepPolicyAPI) List() {
	name := pa.GetString("name")
	projectIDStr := pa.GetString("project_id")
	targetIDStr := pa.GetString("target_id")

	var projectID int64
	var targetID int64
	var err error

	if len(projectIDStr) != 0 {
		projectID, err = strconv.ParseInt(projectIDStr, 10, 64)
		if err != nil || projectID <= 0 {
			pa.CustomAbort(http.StatusBadRequest, "invalid project ID")
		}
	}

	if len(targetIDStr) != 0 {
		targetID, err = strconv.ParseInt(targetIDStr, 10, 64)
		if err != nil || targetID <= 0 {
			pa.CustomAbort(http.StatusBadRequest, "invalid target ID")
		}
	}

	policies, err := dao.FilterRepPolicies(name, projectID, targetID)
	if err != nil {
		log.Errorf("failed to filter policies %s project ID %d: %v", name, projectID, err)
		pa.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
	pa.Data["json"] = policies
	pa.ServeJSON()
}

// Post creates a policy, and if it is enbled, the replication will be triggered right now.
func (pa *RepPolicyAPI) Post() {
	policy := &models.RepPolicy{}
	pa.DecodeJSONReqAndValidate(policy)

	/*
		po, err := dao.GetRepPolicyByName(policy.Name)
		if err != nil {
			log.Errorf("failed to get policy %s: %v", policy.Name, err)
			pa.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		}

		if po != nil {
			pa.CustomAbort(http.StatusConflict, "name is already used")
		}
	*/

	project, err := dao.GetProjectByID(policy.ProjectID)
	if err != nil {
		log.Errorf("failed to get project %d: %v", policy.ProjectID, err)
		pa.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	if project == nil {
		pa.CustomAbort(http.StatusBadRequest, fmt.Sprintf("project %d does not exist", policy.ProjectID))
	}

	target, err := dao.GetRepTarget(policy.TargetID)
	if err != nil {
		log.Errorf("failed to get target %d: %v", policy.TargetID, err)
		pa.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	if target == nil {
		pa.CustomAbort(http.StatusBadRequest, fmt.Sprintf("target %d does not exist", policy.TargetID))
	}

	policies, err := dao.GetRepPolicyByProjectAndTarget(policy.ProjectID, policy.TargetID)
	if err != nil {
		log.Errorf("failed to get policy [project ID: %d,targetID: %d]: %v", policy.ProjectID, policy.TargetID, err)
		pa.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	if len(policies) > 0 {
		pa.CustomAbort(http.StatusConflict, "policy already exists with the same project and target")
	}

	pid, err := dao.AddRepPolicy(*policy)
	if err != nil {
		log.Errorf("Failed to add policy to DB, error: %v", err)
		pa.RenderError(http.StatusInternalServerError, "Internal Error")
		return
	}

	if policy.Enabled == 1 {
		go func() {
			if err := TriggerReplication(pid, "", nil, models.RepOpTransfer); err != nil {
				log.Errorf("failed to trigger replication of %d: %v", pid, err)
			} else {
				log.Infof("replication of %d triggered", pid)
			}
		}()
	}

	pa.Redirect(http.StatusCreated, strconv.FormatInt(pid, 10))
}

// Put modifies name, description, target and enablement of policy
func (pa *RepPolicyAPI) Put() {
	id := pa.GetIDFromURL()
	originalPolicy, err := dao.GetRepPolicy(id)
	if err != nil {
		log.Errorf("failed to get policy %d: %v", id, err)
		pa.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	if originalPolicy == nil {
		pa.CustomAbort(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	}

	policy := &models.RepPolicy{}
	pa.DecodeJSONReq(policy)
	policy.ProjectID = originalPolicy.ProjectID
	pa.Validate(policy)

	/*
		// check duplicate name
		if policy.Name != originalPolicy.Name {
			po, err := dao.GetRepPolicyByName(policy.Name)
			if err != nil {
				log.Errorf("failed to get policy %s: %v", policy.Name, err)
				pa.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			}

			if po != nil {
				pa.CustomAbort(http.StatusConflict, "name is already used")
			}
		}
	*/

	if policy.TargetID != originalPolicy.TargetID {
		//target of policy can not be modified when the policy is enabled
		if originalPolicy.Enabled == 1 {
			pa.CustomAbort(http.StatusBadRequest, "target of policy can not be modified when the policy is enabled")
		}

		// check the existance of target
		target, err := dao.GetRepTarget(policy.TargetID)
		if err != nil {
			log.Errorf("failed to get target %d: %v", policy.TargetID, err)
			pa.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		}

		if target == nil {
			pa.CustomAbort(http.StatusBadRequest, fmt.Sprintf("target %d does not exist", policy.TargetID))
		}

		// check duplicate policy with the same project and target
		policies, err := dao.GetRepPolicyByProjectAndTarget(policy.ProjectID, policy.TargetID)
		if err != nil {
			log.Errorf("failed to get policy [project ID: %d,targetID: %d]: %v", policy.ProjectID, policy.TargetID, err)
			pa.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		}

		if len(policies) > 0 {
			pa.CustomAbort(http.StatusConflict, "policy already exists with the same project and target")
		}
	}

	policy.ID = id

	/*
		isTargetChanged := !(policy.TargetID == originalPolicy.TargetID)
		isEnablementChanged := !(policy.Enabled == policy.Enabled)

		var shouldStop, shouldTrigger bool

		// if target and enablement are not changed, do nothing
		if !isTargetChanged && !isEnablementChanged {
			shouldStop = false
			shouldTrigger = false
		} else if !isTargetChanged && isEnablementChanged {
			// target is not changed, but enablement is changed
			if policy.Enabled == 0 {
				shouldStop = true
				shouldTrigger = false
			} else {
				shouldStop = false
				shouldTrigger = true
			}
		} else if isTargetChanged && !isEnablementChanged {
			// target is changed, but enablement is not changed
			if policy.Enabled == 0 {
				// enablement is 0, do nothing
				shouldStop = false
				shouldTrigger = false
			} else {
				// enablement is 1, so stop original target's jobs
				// and trigger new target's jobs
				shouldStop = true
				shouldTrigger = true
			}
		} else {
			// both target and enablement are changed

			// enablement: 1 -> 0
			if policy.Enabled == 0 {
				shouldStop = true
				shouldTrigger = false
			} else {
				shouldStop = false
				shouldTrigger = true
			}
		}

		if shouldStop {
			if err := postReplicationAction(id, "stop"); err != nil {
				log.Errorf("failed to stop replication of %d: %v", id, err)
				pa.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			}
			log.Infof("replication of %d has been stopped", id)
		}

		if err = dao.UpdateRepPolicy(policy); err != nil {
			log.Errorf("failed to update policy %d: %v", id, err)
			pa.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		}

		if shouldTrigger {
			go func() {
				if err := TriggerReplication(id, "", nil, models.RepOpTransfer); err != nil {
					log.Errorf("failed to trigger replication of %d: %v", id, err)
				} else {
					log.Infof("replication of %d triggered", id)
				}
			}()
		}
	*/

	if err = dao.UpdateRepPolicy(policy); err != nil {
		log.Errorf("failed to update policy %d: %v", id, err)
		pa.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	if policy.Enabled != originalPolicy.Enabled && policy.Enabled == 1 {
		go func() {
			if err := TriggerReplication(id, "", nil, models.RepOpTransfer); err != nil {
				log.Errorf("failed to trigger replication of %d: %v", id, err)
			} else {
				log.Infof("replication of %d triggered", id)
			}
		}()
	}
}

type enablementReq struct {
	Enabled int `json:"enabled"`
}

// UpdateEnablement changes the enablement of the policy
func (pa *RepPolicyAPI) UpdateEnablement() {
	id := pa.GetIDFromURL()
	policy, err := dao.GetRepPolicy(id)
	if err != nil {
		log.Errorf("failed to get policy %d: %v", id, err)
		pa.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
	//@lili
	if policy.Partial == 1 {
		pa.CustomAbort(http.StatusBadRequest, "invalid policy")
		return
	}

	if policy == nil {
		pa.CustomAbort(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	}

	e := enablementReq{}
	pa.DecodeJSONReq(&e)
	if e.Enabled != 0 && e.Enabled != 1 {
		pa.RenderError(http.StatusBadRequest, "invalid enabled value")
		return
	}

	if policy.Enabled == e.Enabled {
		return
	}

	if err := dao.UpdateRepPolicyEnablement(id, e.Enabled); err != nil {
		log.Errorf("Failed to update policy enablement in DB, error: %v", err)
		pa.RenderError(http.StatusInternalServerError, "Internal Error")
		return
	}

	if e.Enabled == 1 {
		go func() {
			if err := TriggerReplication(id, "", nil, models.RepOpTransfer); err != nil {
				log.Errorf("failed to trigger replication of %d: %v", id, err)
			} else {
				log.Infof("replication of %d triggered", id)
			}
		}()
	} else {
		go func() {
			if err := postReplicationAction(id, "stop"); err != nil {
				log.Errorf("failed to stop replication of %d: %v", id, err)
			} else {
				log.Infof("try to stop replication of %d", id)
			}
		}()
	}
}

// Delete : policies which are disabled and have no running jobs
// can be deleted
func (pa *RepPolicyAPI) Delete() {
	id := pa.GetIDFromURL()
	policy, err := dao.GetRepPolicy(id)
	if err != nil {
		log.Errorf("failed to get policy %d: %v", id, err)
		pa.CustomAbort(http.StatusInternalServerError, "")
	}

	if policy == nil || policy.Deleted == 1 {
		pa.CustomAbort(http.StatusNotFound, "")
	}

	if policy.Enabled == 1 {
		//@lili
		if policy.Partial == 0 {
			pa.CustomAbort(http.StatusPreconditionFailed, "plicy is enabled, can not be deleted")
		}

	}

	jobs, err := dao.GetRepJobByPolicy(id)
	if err != nil {
		log.Errorf("failed to get jobs of policy %d: %v", id, err)
		pa.CustomAbort(http.StatusInternalServerError, "")
	}

	for _, job := range jobs {
		if job.Status == models.JobRunning ||
			job.Status == models.JobRetrying ||
			job.Status == models.JobPending {
			pa.CustomAbort(http.StatusPreconditionFailed, "policy has running/retrying/pending jobs, can not be deleted")
		}
	}

	if err = dao.DeleteRepPolicy(id); err != nil {
		log.Errorf("failed to delete policy %d: %v", id, err)
		pa.CustomAbort(http.StatusInternalServerError, "")
	}
}

// Post creates a policy, and if it is enbled, the replication will be triggered right now.
func (pa *RepPolicyAPI) PartialReplication() {
	policy := &models.PartialReplicationPolicy{}
	pa.DecodeJSONReqAndValidate(policy)
	//set enabled=-1 ,marked as PartialReplication
	policy.Enabled = 1
	policy.Partial = 1
	project, err := dao.GetProjectByID(policy.ProjectID)
	if err != nil {
		log.Errorf("failed to get project %d: %v", policy.ProjectID, err)
		pa.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	if project == nil {
		pa.CustomAbort(http.StatusBadRequest, fmt.Sprintf("project %d does not exist", policy.ProjectID))
	}

	target, err := dao.GetRepTarget(policy.TargetID)
	if err != nil {
		log.Errorf("failed to get target %d: %v", policy.TargetID, err)
		pa.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	if target == nil {
		pa.CustomAbort(http.StatusBadRequest, fmt.Sprintf("target %d does not exist", policy.TargetID))
	}

	policies, err := dao.GetRepPolicyByProjectAndTarget(policy.ProjectID, policy.TargetID)
	if err != nil {
		log.Errorf("failed to get policy [project ID: %d,targetID: %d]: %v", policy.ProjectID, policy.TargetID, err)
		pa.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
	var pid int64
	if len(policies) > 0 {
		//check policy existing
		if policies[0].Partial == 0 {
			pa.CustomAbort(http.StatusConflict, "policy already exists with the same project and target")
		} else {
			pid = policies[0].ID
			err = dao.UpdatePolicyTime(pid)
			if err != nil {
				log.Errorf("Failed to update policyTime to DB, error: %v", err)
				pa.RenderError(http.StatusInternalServerError, "Internal Error")
				return
			}
		}

	} else {
		//check here again
		pid, err = dao.AddRepPolicy(policy.RepPolicy)
		if err != nil {
			log.Errorf("Failed to add policy to DB, error: %v", err)
			pa.RenderError(http.StatusInternalServerError, "Internal Error")
			return
		}
	}

	//	if policy.Enabled == 1 {
	//var tags []string
	ch := make(chan int, 1)
	for repo, _ := range policy.Image {
		tags := policy.Image[repo]
		//log.Infof("***************repo***************%s", repo)
		go func() {
			ch <- 1
			//log.Infof("***************start***************%s", repo)
			if err := TriggerReplication(pid, repo, tags, models.RepOpTransfer); err != nil {
				log.Errorf("failed to trigger replication of %d: %v", pid, err)
			} else {
				log.Infof("replication of %d triggered", pid)
			}
		}()
		select {
		case <-ch:
		case <-time.After(10 * time.Second):
			log.Errorf("failed to start current replication,repoNmae%s", repo)
			continue
			//panic("Failed to start after 10 seconds")
		}
		//log.Infof("***************end***************%s", repo)
	}
	//	}

	pa.Redirect(http.StatusCreated, strconv.FormatInt(pid, 10))
}

type UnfinishedNum struct {
	UnfinNum int64 `json:"unfinishedNum"`
	TotalNum int64 `json:"totalNum"`
	ErrorNum int64 `json:"errorNum"`
	//test          string `json:"test"`
}

func (pa *RepPolicyAPI) GetStatus() {
	id := pa.GetIDFromURL()
	//	var number int64
	// for test
	//number = 100
	var runningNum int64
	var totalNum int64
	var errorNum int64
	policy, err := dao.GetRepPolicy(id)
	if err != nil {
		log.Errorf("failed to get policy %d: %v", id, err)
		pa.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	if policy == nil {
		pa.CustomAbort(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	}

	updateTime, err := dao.GetPolicyUpdateTime(id)
	if err != nil {
		log.Errorf("Failed to get status of policy, error: %v", err)
		pa.RenderError(http.StatusInternalServerError, "Internal Error e1")
		return
	}
	runningNum, err = dao.GetPolicyRunningJob(id, updateTime)
	if err != nil {
		log.Errorf("Failed to get status of policy, error: %v", err)
		pa.RenderError(http.StatusInternalServerError, "Internal Error e2")
		return
	}
	totalNum, err = dao.GetPolicyCurrentJob(id, updateTime)
	if err != nil {
		log.Errorf("Failed to get status of policy, error: %v", err)
		pa.RenderError(http.StatusInternalServerError, "Internal Error e3")
		return
	}
	errorNum, err = dao.GetPolicyErrorJob(id, updateTime)
	if err != nil {
		log.Errorf("Failed to get status of policy, error: %v", err)
		pa.RenderError(http.StatusInternalServerError, "Internal Error e3")
		return
	}
	/*
		number, err = dao.GetRepPolicyStatus(id)
		if err != nil {
			log.Errorf("Failed to get status of policy, error: %v", err)
			pa.RenderError(http.StatusInternalServerError, "Internal Error")
			return
		}
	*/
	nfn := UnfinishedNum{}
	nfn.UnfinNum = runningNum
	nfn.TotalNum = totalNum
	nfn.ErrorNum = errorNum
	pa.Data["json"] = nfn
	pa.ServeJSON()
}
