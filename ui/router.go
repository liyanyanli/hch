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

package main

import (
	"github.com/vmware/harbor/api"
	"github.com/vmware/harbor/controllers"
	"github.com/vmware/harbor/service"
	"github.com/vmware/harbor/service/token"

	"github.com/astaxie/beego"
)

func initRouters() {

	beego.SetStaticPath("static/resources", "static/resources")
	beego.SetStaticPath("static/vendors", "static/vendors")

	//Page Controllers:
	beego.Router("/", &controllers.IndexController{})
	beego.Router("/dashboard", &controllers.DashboardController{})
	beego.Router("/project", &controllers.ProjectController{})
	beego.Router("/repository", &controllers.RepositoryController{})
	beego.Router("/sign_up", &controllers.SignUpController{})
	beego.Router("/add_new", &controllers.AddNewController{})
	beego.Router("/account_setting", &controllers.AccountSettingController{})
	beego.Router("/change_password", &controllers.ChangePasswordController{})
	beego.Router("/admin_option", &controllers.AdminOptionController{})
	beego.Router("/forgot_password", &controllers.ForgotPasswordController{})
	beego.Router("/reset_password", &controllers.ResetPasswordController{})
	beego.Router("/search", &controllers.SearchController{})

	beego.Router("/login", &controllers.CommonController{}, "post:Login")
	beego.Router("/log_out", &controllers.CommonController{}, "get:LogOut")
	beego.Router("/reset", &controllers.CommonController{}, "post:ResetPassword")
	beego.Router("/userExists", &controllers.CommonController{}, "post:UserExists")
	beego.Router("/sendEmail", &controllers.CommonController{}, "get:SendEmail")
	beego.Router("/language", &controllers.CommonController{}, "get:SwitchLanguage")

	beego.Router("/optional_menu", &controllers.OptionalMenuController{})
	beego.Router("/navigation_header", &controllers.NavigationHeaderController{})
	beego.Router("/navigation_detail", &controllers.NavigationDetailController{})
	beego.Router("/sign_in", &controllers.SignInController{})

	//API:
	beego.Router("/api/search", &api.SearchAPI{})
	beego.Router("/api/projects/:pid([0-9]+)/members/?:mid", &api.ProjectMemberAPI{})
	beego.Router("/api/projects/", &api.ProjectAPI{}, "get:List;post:Post")
	beego.Router("/api/projects/:id", &api.ProjectAPI{})
	beego.Router("/api/projects/:id/publicity", &api.ProjectAPI{}, "put:ToggleProjectPublic")
	//@lili
	beego.Router("/api/projects/:id/quota", &api.ProjectAPI{}, "post:UpdateProjectQuota")
	beego.Router("/api/projects/quotaList", &api.ProjectAPI{}, "get:GetProjectQuota")

	beego.Router("/api/statistics", &api.StatisticAPI{})
	beego.Router("/api/projects/:id([0-9]+)/logs/filter", &api.ProjectAPI{}, "post:FilterAccessLog")
	beego.Router("/api/users/?:id", &api.UserAPI{})
	beego.Router("/api/users/:id([0-9]+)/password", &api.UserAPI{}, "put:ChangePassword")
	beego.Router("/api/internal/syncregistry", &api.InternalAPI{}, "post:SyncRegistry")
	beego.Router("/api/repositories", &api.RepositoryAPI{})
	beego.Router("/api/repositories/tags", &api.RepositoryAPI{}, "get:GetTags")
	beego.Router("/api/repositories/manifests", &api.RepositoryAPI{}, "get:GetManifests")
	beego.Router("/api/jobs/replication/", &api.RepJobAPI{}, "get:List")
	beego.Router("/api/jobs/replication/:id([0-9]+)", &api.RepJobAPI{})
	beego.Router("/api/jobs/replication/:id([0-9]+)/log", &api.RepJobAPI{}, "get:GetLog")
	beego.Router("/api/policies/replication/:id([0-9]+)", &api.RepPolicyAPI{})
	beego.Router("/api/policies/replication", &api.RepPolicyAPI{}, "get:List")
	beego.Router("/api/policies/partialreplication", &api.RepPolicyAPI{}, "post:PartialReplication")
	beego.Router("/api/policies/replication", &api.RepPolicyAPI{}, "post:Post")
	beego.Router("/api/policies/replication/:id([0-9]+)/enablement", &api.RepPolicyAPI{}, "put:UpdateEnablement")
	beego.Router("/api/targets/", &api.TargetAPI{}, "get:List")
	beego.Router("/api/targets/", &api.TargetAPI{}, "post:Post")
	beego.Router("/api/targets/:id([0-9]+)", &api.TargetAPI{})
	beego.Router("/api/targets/:id([0-9]+)/policies/", &api.TargetAPI{}, "get:ListPolicies")
	beego.Router("/api/targets/ping", &api.TargetAPI{}, "post:Ping")
	beego.Router("/api/users/:id/sysadmin", &api.UserAPI{}, "put:ToggleUserAdminRole")
	beego.Router("/api/repositories/top", &api.RepositoryAPI{}, "get:GetTopRepos")
	beego.Router("/api/logs", &api.LogAPI{})
	beego.Router("/api/repositories/getClairStatistcs", &api.RepositoryAPI{}, "get:GetClairStatistcs")
	beego.Router("/api/repositories/getVulnerabilitiesByPackage", &api.RepositoryAPI{}, "get:GetPackageSummary")
	beego.Router("/api/repositories/getVulnerabilitySummary", &api.RepositoryAPI{}, "get:GetVulnerabilitySummary")
	beego.Router("/api/repositories/copyImage", &api.RepositoryAPI{}, "post:CopyRepo")
	beego.Router("/api/repositories/copyProjects", &api.RepositoryAPI{}, "post:CopyPro")
	beego.Router("/api/policies/replication/:id([0-9]+)/status", &api.RepPolicyAPI{}, "get:GetStatus")
	//resource recycle interface:
	beego.Router("/api/image/getImageList", &api.ImageAPI{}, "get:GetImageList")
	beego.Router("/api/image/extendImage", &api.ImageAPI{}, "post:ExtendImage")
	beego.Router("/api/adminManage", &api.AdminManage{}, "get:UpdateAdminTime")

	beego.Router("/api/scanImages", &api.ScanImagesAPI{}, "post:ScanImages")

	beego.Router("/api/initCVEStatus", &api.ScanImagesAPI{}, "post:InitCVEStatus")
	//external service that hosted on harbor process:
	beego.Router("/service/notifications", &service.NotificationHandler{})
	beego.Router("/service/token", &token.Handler{})

	//interface for host security
	beego.Router("/api/benchSecurity", &api.BenchSecurityAPI{}, "post:SaveBenchSecurity")
	beego.Router("/api/getBenchSecurity", &api.BenchSecurityAPI{}, "get:GetBenchSecurity")
	beego.Router("/api/getRuntimeByIp", &api.BenchSecurityAPI{}, "get:GetRuntimeByIp")

	beego.Router("/api/benchSecurityRuntime", &api.BenchSecurityAPI{}, "post:SaveBenchSecurityRuntime")
	beego.Router("/api/getBenchSecurityRuntime", &api.BenchSecurityAPI{}, "get:GetBenchSecurityRuntime")

}
