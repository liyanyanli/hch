package api

import (
	"net/http"

	"github.com/vmware/harbor/dao"
	"github.com/vmware/harbor/utils/log"
)

type AdminManage struct {
	BaseAPI
	uID int
}

//Prepare validates the URL and the user
func (am *AdminManage) Prepare() {
	am.uID = am.ValidateUser()
}

func (am *AdminManage) UpdateAdminTime() {
	const DT = `admin_detectTime`
	const CT = `admin_cycleTime`
	dt, _ := am.GetInt("detect_time")
	ct, _ := am.GetInt("cycle_time")
	if dt == 0 && ct == 0 {
		am.CustomAbort(http.StatusBadRequest, "time is null")
	}

	var err error
	isAdmin := false
	am.uID = am.ValidateUser()
	isAdmin, err = dao.IsAdminRole(am.uID)
	if err != nil {
		log.Errorf("Error occured in check admin, error: %v", err)
		am.CustomAbort(http.StatusInternalServerError, "Internal error.")
	}

	if isAdmin {
		if dt != 0 {
			err := dao.UpdateAdminTime(DT, dt)
			if err != nil {
				log.Errorf("Error happens when update admin_detectTime : %v", err)
				am.CustomAbort(http.StatusInternalServerError, "Error happens when update admin_detectTime from sql")
			}
		}

		if ct != 0 {
			err := dao.UpdateAdminTime(CT, ct)
			if err != nil {
				log.Errorf("Error happens when update admin_cycleTime : %v", err)
				am.CustomAbort(http.StatusInternalServerError, "Error happens when update admin_cycleTime from sql")
			}
		}
	} else {
		am.CustomAbort(http.StatusBadRequest, "The current user does not have admin privileges.")
	}
}
