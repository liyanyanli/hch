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
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	//"github.com/vmware/harbor/api"
	"github.com/vmware/harbor/dao"
	"github.com/vmware/harbor/models"
	"github.com/vmware/harbor/service/cache"
	"github.com/vmware/harbor/service/token"
	svc_utils "github.com/vmware/harbor/service/utils"
	"github.com/vmware/harbor/utils/log"
	"github.com/vmware/harbor/utils/registry"

	registry_error "github.com/vmware/harbor/utils/registry/error"

	"reflect"

	"github.com/vmware/harbor/utils"
	"github.com/vmware/harbor/utils/registry/auth"
	"strings"
	"regexp"
	"strconv"
)

// RepositoryAPI handles request to /api/repositories /api/repositories/tags /api/repositories/manifests, the parm has to be put
// in the query string as the web framework can not parse the URL if it contains veriadic sectors.
type RepositoryAPI struct {
	BaseAPI
}

// Get ...
func (ra *RepositoryAPI) Get() {
	projectID, err := ra.GetInt64("project_id")
	if err != nil || projectID <= 0 {
		ra.CustomAbort(http.StatusBadRequest, "invalid project_id")
	}

	page, pageSize := ra.getPaginationParams()

	project, err := dao.GetProjectByID(projectID)
	if err != nil {
		log.Errorf("failed to get project %d: %v", projectID, err)
		ra.CustomAbort(http.StatusInternalServerError, "")
	}

	if project == nil {
		ra.CustomAbort(http.StatusNotFound, fmt.Sprintf("project %d not found", projectID))
	}

	if project.Public == 0 {
		var userID int

		if svc_utils.VerifySecret(ra.Ctx.Request) {
			userID = 1
		} else {
			userID = ra.ValidateUser()
		}

		if !checkProjectPermission(userID, projectID) {
			ra.CustomAbort(http.StatusForbidden, "")
		}
	}

	repositories, err := getReposByProject(project.Name, ra.GetString("q"))
	if err != nil {
		log.Errorf("failed to get repository: %v", err)
		ra.CustomAbort(http.StatusInternalServerError, "")
	}

	total := int64(len(repositories))

	if (page-1)*pageSize > total {
		repositories = []string{}
	} else {
		repositories = repositories[(page-1)*pageSize:]
	}

	if page*pageSize <= total {
		repositories = repositories[:pageSize]
	}

	ra.setPaginationHeader(total, page, pageSize)

	ra.Data["json"] = repositories
	ra.ServeJSON()
}

// Delete ...
func (ra *RepositoryAPI) Delete() {
	repoName := ra.GetString("repo_name")
	if len(repoName) == 0 {
		ra.CustomAbort(http.StatusBadRequest, "repo_name is nil")
	}

	projectName, _ := utils.ParseRepository(repoName)
	project, err := dao.GetProjectByName(projectName)
	if err != nil {
		log.Errorf("failed to get project %s: %v", projectName, err)
		ra.CustomAbort(http.StatusInternalServerError, "")
	}

	if project == nil {
		ra.CustomAbort(http.StatusNotFound, fmt.Sprintf("project %s not found", projectName))
	}

	if project.Public == 0 {
		userID := ra.ValidateUser()
		if !hasProjectAdminRole(userID, project.ProjectID) {
			ra.CustomAbort(http.StatusForbidden, "")
		}
	}

	rc, err := ra.initRepositoryClient(repoName)
	if err != nil {
		log.Errorf("error occurred while initializing repository client for %s: %v", repoName, err)
		ra.CustomAbort(http.StatusInternalServerError, "internal error")
	}

	tags := []string{}
	tag := ra.GetString("tag")
	if len(tag) == 0 {
		tagList, err := rc.ListTag()
		if err != nil {
			if regErr, ok := err.(*registry_error.Error); ok {
				ra.CustomAbort(regErr.StatusCode, regErr.Detail)
			}

			log.Errorf("error occurred while listing tags of %s: %v", repoName, err)
			ra.CustomAbort(http.StatusInternalServerError, "internal error")
		}

		// TODO remove the logic if the bug of registry is fixed
		if len(tagList) == 0 {
			//ra.CustomAbort(http.StatusNotFound, http.StatusText(http.StatusNotFound))
			log.Errorf("not found tags of %s", repoName)
		}

		tags = append(tags, tagList...)
	} else {
		if strings.Contains(tag,",") {
			multipleTags := strings.Split(tag, ",")
			tags = append(tags, multipleTags...)
		} else {
			tags = append(tags, tag)
		}
	}

	user, _, ok := ra.Ctx.Request.BasicAuth()
	if !ok {
		user, err = ra.getUsername()
		if err != nil {
			log.Errorf("failed to get user: %v", err)
		}
	}

	for _, t := range tags {

		notFoundFlag := false
		if err := rc.DeleteTag(t); err != nil {
			if regErr, ok := err.(*registry_error.Error); ok {
				if regErr.StatusCode != http.StatusNotFound {
					ra.CustomAbort(regErr.StatusCode, regErr.Detail)
				} else {
					notFoundFlag = true;
					log.Errorf("PPPPPPPPPPPPPPPPPPPPPPPPPPPPPPPPPP%s:%s", repoName, t)
				}
			} else {
				log.Errorf("error occurred while deleting tag %s:%s: %v", repoName, t, err)
				ra.CustomAbort(http.StatusInternalServerError, "internal error")
			}
		}
		imageName := repoName + ":" + t
		log.Info("&&&&&&&&&&***********imageName %s", imageName)
		updatedQuotaFlag := ra.UpdateQuota(projectName, imageName);
		log.Info("update project %s quota result %t", imageName, updatedQuotaFlag)
		//tag not found and update quota fail, skip and  go to next tag
		if notFoundFlag && !updatedQuotaFlag {
			log.Errorf("fail to delete tag: %s:%s, tag not found and update quota error.", repoName, t)
			continue;
		}

		//@lili
		log.Infof("delete tag: %s:%s", repoName, t)
		go TriggerReplicationByRepository(repoName, []string{t}, models.RepOpDelete)

		go func(tag string) {
			if err := dao.AccessLog(user, projectName, repoName, tag, "delete"); err != nil {
				log.Errorf("failed to add access log: %v", err)
			}
		}(t)
	}

	exist, err := repositoryExist(repoName, rc)
	if err != nil {
		log.Errorf("failed to check the existence of repository %s: %v", repoName, err)
		ra.CustomAbort(http.StatusInternalServerError, "")
	}
	if !exist {
		if err = dao.DeleteRepository(repoName); err != nil {
			log.Errorf("failed to delete repository %s: %v", repoName, err)
			ra.CustomAbort(http.StatusInternalServerError, "")
		}
	}

	go func() {
		log.Debug("refreshing catalog cache")
		if err := cache.RefreshCatalogCache(); err != nil {
			log.Errorf("error occurred while refresh catalog cache: %v", err)
		}
	}()
}

func (ra *RepositoryAPI) UpdateQuota(projectName string, imageName string) (bool){
	useSize, erro := dao.GetUseSizeByProject(projectName)
	if erro != nil {
		log.Errorf("failed to get the use size from project_quota %s: %v", projectName, erro)
		return false;
	}
	imageSize, erro := dao.GetSizeOfImage(imageName)
	if erro != nil {
		log.Errorf("failed to get the image size from project_quota %s: %v", imageName, erro)
		return false;
	}
	if imageSize <= 0 {
		log.Errorf("failed to get the image size from project_quota %s, size %f", imageName, imageSize)
		return false;
	}
	log.Info("update quota for imageName %s, image size %f, use size %f", imageName, imageSize, useSize)
	erro = dao.UpdateUseQuotaForDelete(projectName, imageName)
	if erro != nil {
		log.Errorf("failed to update  use quota from project_quota %s: %v", projectName, erro)
		return false;
	}
	erro = dao.DelectImage(imageName)
	if erro != nil {
		log.Errorf("failed to delete image from clari_result %s: %v", imageName, erro)
	}
	return true;
}

type tag struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type imgMassage struct {
	Tag       string `json:"tag"`
	HighNum   int    `json:"high_num"`
	OtherNum  int    `json:"other_num"`
	CVEStatus int    `json:"CVE_status"`
	ScanS     bool   `json:"scan_statue"`
}

// GetTags handles GET /api/repositories/tags
func (ra *RepositoryAPI) GetTags() {
	ra.ValidateUser()
	repoName := ra.GetString("repo_name")
	if len(repoName) == 0 {
		ra.CustomAbort(http.StatusBadRequest, "repo_name is nil")
	}

	projectName, _ := utils.ParseRepository(repoName)
	project, err := dao.GetProjectByName(projectName)
	if err != nil {
		log.Errorf("failed to get project %s: %v", projectName, err)
		ra.CustomAbort(http.StatusInternalServerError, "")
	}

	if project == nil {
		ra.CustomAbort(http.StatusNotFound, fmt.Sprintf("project %s not found", projectName))
	}

	if project.Public == 0 {
		userID := ra.ValidateUser()
		if !checkProjectPermission(userID, project.ProjectID) {
			ra.CustomAbort(http.StatusForbidden, "")
		}
	}

	rc, err := ra.initRepositoryClient(repoName)

	if err != nil {
		log.Errorf("error occurred while initializing repository client for %s: %v", repoName, err)
		ra.CustomAbort(http.StatusInternalServerError, "internal error")
	}

	tags := []string{}

	ts, err := rc.ListTag()

	if err != nil {
		regErr, ok := err.(*registry_error.Error)
		if !ok {
			log.Errorf("error occurred while listing tags of %s: %v", repoName, err)
			ra.CustomAbort(http.StatusInternalServerError, "internal error")
		}
		// TODO remove the logic if the bug of registry is fixed
		// It's a workaround for a bug of registry: when listing tags of
		// a repository which is being pushed, a "NAME_UNKNOWN" error will
		// been returned, while the catalog API can list this repository.
		if regErr.StatusCode != http.StatusNotFound {
			ra.CustomAbort(regErr.StatusCode, regErr.Detail)
		}
	}

	tags = append(tags, ts...)

	sort.Strings(tags)

	var imageM imgMassage

	imageMs := []imgMassage{}

	for _, tag := range tags {

		var repo_name = repoName + ":" + tag

		highNum, err := dao.GetHighNumByRepoName(repo_name)
		if err != nil {
			log.Errorf("failed to num erro: %v", err)
		}

		otherNum, err := dao.GetOtherNumByRepoName(repo_name)
		if err != nil {
			log.Errorf("failed to num erro: %v ", err)
		}

		cveStatus, err := dao.GetCVEStatusByRepoName(repo_name)
		otherStatue, err := dao.GetOtherStatueByRepoName(repo_name)

		if err != nil {
			log.Errorf("failed to num erro: %v ", err)
		}

		if otherStatue == 2 || otherStatue == 4 || otherStatue == 5 {
			imageM.ScanS = true
		} else {
			imageM.ScanS = false
		}

		imageM.Tag = tag
		imageM.HighNum = highNum
		imageM.OtherNum = otherNum
		imageM.CVEStatus = cveStatus
		imageMs = append(imageMs, imageM)
	}

	ra.Data["json"] = imageMs
	ra.ServeJSON()
}

// GetManifests handles GET /api/repositories/manifests
func (ra *RepositoryAPI) GetManifests() {
	repoName := ra.GetString("repo_name")
	tag := ra.GetString("tag")

	if len(repoName) == 0 || len(tag) == 0 {
		ra.CustomAbort(http.StatusBadRequest, "repo_name or tag is nil")
	}

	version := ra.GetString("version")
	if len(version) == 0 {
		version = "v2"
	}

	if version != "v1" && version != "v2" {
		ra.CustomAbort(http.StatusBadRequest, "version should be v1 or v2")
	}

	projectName, _ := utils.ParseRepository(repoName)
	project, err := dao.GetProjectByName(projectName)
	if err != nil {
		log.Errorf("failed to get project %s: %v", projectName, err)
		ra.CustomAbort(http.StatusInternalServerError, "")
	}

	if project == nil {
		ra.CustomAbort(http.StatusNotFound, fmt.Sprintf("project %s not found", projectName))
	}

	if project.Public == 0 {
		userID := ra.ValidateUser()
		if !checkProjectPermission(userID, project.ProjectID) {
			ra.CustomAbort(http.StatusForbidden, "")
		}
	}

	rc, err := ra.initRepositoryClient(repoName)
	if err != nil {
		log.Errorf("error occurred while initializing repository client for %s: %v", repoName, err)
		ra.CustomAbort(http.StatusInternalServerError, "internal error")
	}

	result := struct {
		Manifest interface{} `json:"manifest"`
		Config   interface{} `json:"config,omitempty" `
	}{}

	mediaTypes := []string{}
	switch version {
	case "v1":
		mediaTypes = append(mediaTypes, schema1.MediaTypeManifest)
	case "v2":
		mediaTypes = append(mediaTypes, schema2.MediaTypeManifest)
	}

	_, mediaType, payload, err := rc.PullManifest(tag, mediaTypes)
	if err != nil {
		if regErr, ok := err.(*registry_error.Error); ok {
			ra.CustomAbort(regErr.StatusCode, regErr.Detail)
		}

		log.Errorf("error occurred while getting manifest of %s:%s: %v", repoName, tag, err)
		ra.CustomAbort(http.StatusInternalServerError, "internal error")
	}

	manifest, _, err := registry.UnMarshal(mediaType, payload)
	if err != nil {
		log.Errorf("an error occurred while parsing manifest of %s:%s: %v", repoName, tag, err)
		ra.CustomAbort(http.StatusInternalServerError, "")
	}

	result.Manifest = manifest

	deserializedmanifest, ok := manifest.(*schema2.DeserializedManifest)
	if ok {
		_, data, err := rc.PullBlob(deserializedmanifest.Target().Digest.String())
		if err != nil {
			log.Errorf("failed to get config of manifest %s:%s: %v", repoName, tag, err)
			ra.CustomAbort(http.StatusInternalServerError, "")
		}

		b, err := ioutil.ReadAll(data)
		if err != nil {
			log.Errorf("failed to read config of manifest %s:%s: %v", repoName, tag, err)
			ra.CustomAbort(http.StatusInternalServerError, "")
		}

		result.Config = string(b)
	}

	ra.Data["json"] = result
	ra.ServeJSON()
}

func (ra *RepositoryAPI) initRepositoryClient(repoName string) (r *registry.Repository, err error) {
	endpoint := os.Getenv("REGISTRY_URL")

	username, password, ok := ra.Ctx.Request.BasicAuth()
	if ok {
		return newRepositoryClient(endpoint, getIsInsecure(), username, password,
			repoName, "repository", repoName, "pull", "push", "*")
	}

	username, err = ra.getUsername()
	if err != nil {
		return nil, err
	}

	return cache.NewRepositoryClient(endpoint, getIsInsecure(), username, repoName,
		"repository", repoName, "pull", "push", "*")
}

func (ra *RepositoryAPI) getUsername() (string, error) {
	// get username from session
	sessionUsername := ra.GetSession("username")
	if sessionUsername != nil {
		username, ok := sessionUsername.(string)
		if ok {
			return username, nil
		}
	}

	// if username does not exist in session, try to get userId from sessiion
	// and then get username from DB according to the userId
	sessionUserID := ra.GetSession("userId")
	if sessionUserID != nil {
		userID, ok := sessionUserID.(int)
		if ok {
			u := models.User{
				UserID: userID,
			}
			user, err := dao.GetUser(u)
			if err != nil {
				return "", err
			}

			return user.Username, nil
		}
	}

	return "", nil
}

//GetTopRepos handles request GET /api/repositories/top
func (ra *RepositoryAPI) GetTopRepos() {
	count, err := ra.GetInt("count", 10)
	if err != nil || count <= 0 {
		ra.CustomAbort(http.StatusBadRequest, "invalid count")
	}

	repos, err := dao.GetTopRepos(count)
	if err != nil {
		log.Errorf("failed to get top repos: %v", err)
		ra.CustomAbort(http.StatusInternalServerError, "internal server error")
	}
	ra.Data["json"] = repos
	ra.ServeJSON()
}

func newRepositoryClient(endpoint string, insecure bool, username, password, repository, scopeType, scopeName string,
	scopeActions ...string) (*registry.Repository, error) {

	credential := auth.NewBasicAuthCredential(username, password)
	authorizer := auth.NewStandardTokenAuthorizer(credential, insecure, scopeType, scopeName, scopeActions...)

	store, err := auth.NewAuthorizerStore(endpoint, insecure, authorizer)
	if err != nil {
		return nil, err
	}

	client, err := registry.NewRepositoryWithModifiers(repository, endpoint, insecure, store)
	if err != nil {
		return nil, err
	}
	return client, nil
}

type clairStatistcs struct {
	ImageNum   int `json:"image_num"`
	UnsaftNum  int `json:"unsecurity_image_num"`
	NotSupport int `json:"clair_not_Support"`
	Success    int `json:"clair_success"`
	Abnormal   int `json:"abnormal"`
	Mild       int `json:"mild"`
}

func (ra *RepositoryAPI) GetClairStatistcs() {
	ra.ValidateUser()
	const PROJECT = `project`
	const USER = `user`
	flagName := ra.GetString("flag_name") //project/user
	name := ra.GetString("name")

	if len(flagName) == 0 || len(name) == 0 {
		ra.CustomAbort(http.StatusBadRequest, "flag_name or name is nil")
	}

	var cs clairStatistcs

	if reflect.DeepEqual(USER, flagName) {
		allTagNum := GetTagNumByUser(name, ra)
		cs = GetStatistcs(cs, allTagNum)

	} else if reflect.DeepEqual(PROJECT, flagName) {
		allTag := GetTagNumByProject(name, ra)
		cs = GetStatistcs(cs, allTag)
	}

	ra.Data["json"] = cs
	ra.ServeJSON()
}

func GetTagNumByRepo(repo string, ra *RepositoryAPI) []string {

	rc, err := ra.initRepositoryClient(repo)

	if err != nil {
		log.Errorf("error occurred while initializing repository client for %s: %v", repo, err)
		ra.CustomAbort(http.StatusInternalServerError, "internal error")
	}

	tags := []string{}
	repoTags := []string{}

	ts, err := rc.ListTag()

	if err != nil {
		regErr, ok := err.(*registry_error.Error)
		if !ok {
			log.Errorf("error occurred while listing tags of %s: %v", repo, err)
			ra.CustomAbort(http.StatusInternalServerError, "internal error")
		}
		// TODO remove the logic if the bug of registry is fixed
		// It's a workaround for a bug of registry: when listing tags of
		// a repository which is being pushed, a "NAME_UNKNOWN" error will
		// been returned, while the catalog API can list this repository.
		if regErr.StatusCode != http.StatusNotFound {
			ra.CustomAbort(regErr.StatusCode, regErr.Detail)
		}
	}

	tags = append(tags, ts...)

	for _, tag := range tags {
		tag = repo + ":" + tag
		repoTags = append(repoTags, tag)
	}

	return repoTags
}

func GetTagNumByProject(projectName string, ra *RepositoryAPI) []string {

	projectSum := []string{}
	projectID, err := dao.GetProjectID(projectName)
	if err != nil {
		log.Errorf("failed to get projectID by projectName error: %v", err)
	}

	repoList, err := dao.GetByProjectId(projectID)
	if err != nil {
		log.Errorf("failed to get repoList by projectID error: %v", err)
	}

	for _, repo := range repoList {
		repoSum := GetTagNumByRepo(repo.Name, ra)
		projectSum = merge(projectSum, repoSum)
	}
	return projectSum

}

func GetTagNumByUser(userName string, ra *RepositoryAPI) []string {

	userSum := []string{}

	userID, err := dao.GetUserID(userName)
	if err != nil {
		log.Errorf("failed to get userID by userName error: %v", err)
	}

	projectList, err := dao.GetProjectByUserID(userID)
	if err != nil {
		log.Errorf("failed to get proList by userID error: %v", err)
	}

	for _, project := range projectList {
		repoSum := GetTagNumByProject(project.Name, ra)
		userSum = merge(userSum, repoSum)
	}

	return userSum
}

func GetStatistcs(cs clairStatistcs, lists []string) clairStatistcs {
	repoLists := "("
	abnormal := 0
	notsupport := 0
	success := 0
	high := 0
	mild := 0
	for _, repoName := range lists {
		if !reflect.DeepEqual(repoLists, "(") {
			repoLists = repoLists + ","
		}
		repoLists = repoLists + "'" + repoName + "'"
	}
	repoLists = repoLists + ")"

	others, err := dao.GetOtherStatusByRepoName(repoLists)
	if err != nil {
		log.Errorf("failed to num erro: %v", err)
	}

	for _, other := range others {
		if other.OtherStatus == 1 {
			notsupport++
		} else if other.OtherStatus == 2 {
			success++
		} else if other.OtherStatus == 4 {
			mild++
		} else if other.OtherStatus == 5 {
			high++
		} else {
			abnormal++
		}
	}
	cs.Abnormal = len(lists) - mild - notsupport - success - high
	cs.ImageNum = len(lists)
	cs.Mild = mild
	cs.NotSupport = notsupport
	cs.Success = success
	cs.UnsaftNum = high

	return cs
}

func merge(a []string, b []string) []string {
	c := make([]string, len(a)+len(b))
	copy(c, a)
	copy(c[len(a):], b)
	return c
}

//CopyRepo handles request Post /api/repositories/copy
///api/repositories/copy
type copyImage struct {
	SrcRepoName  string `json:"src_repo_name"`
	SrcTag       string `json:"src_tag"`
	DestRepoName string `json:"dest_repo_name"`
	DestTag      string `json:"dest_tag"`
}

func (ra *RepositoryAPI) CopyRepo() {
	var ci copyImage

	ra.DecodeJSONReq(&ci)
	srcRepoName := ci.SrcRepoName
	srcTag := ci.SrcTag

	//srcRepoName := ra.GetString("src_repo_name")
	//srcTag := ra.GetString("src_tag")

	if len(srcRepoName) == 0 || len(srcTag) == 0 {
		ra.CustomAbort(http.StatusBadRequest, "Source repo_name or tag is nil")
	}
	//destRepoName := ra.GetString("dest_repo_name")
	//destTag := ra.GetString("dest_tag")
	destRepoName := ci.DestRepoName
	destTag := ci.DestTag
	if len(destTag) == 0 || len(destRepoName) == 0 {
		ra.CustomAbort(http.StatusBadRequest, "Destination repo_name or tag is nil")
	}

	srcProjectName, _ := utils.ParseRepository(srcRepoName)
	srcProject, err := dao.GetProjectByName(srcProjectName)
	if err != nil {
		log.Errorf("failed to get project %s: %v", srcProjectName, err)
		ra.CustomAbort(http.StatusInternalServerError, "")
	}

	if srcProject == nil {
		ra.CustomAbort(http.StatusNotFound, fmt.Sprintf("project %s not found", srcProjectName))
	}

	if srcProject.Public == 0 {
		userID := ra.ValidateUser()
		//check pull authorized
		if !checkProjectPermission(userID, srcProject.ProjectID) {
			ra.CustomAbort(http.StatusForbidden, "unauthorized,operation(pulling) is not authorized")
		}
	}

	destProjectName, _ := utils.ParseRepository(destRepoName)
	destProject, err := dao.GetProjectByName(destProjectName)
	if err != nil {
		log.Errorf("failed to get project %s: %v", destProjectName, err)
		ra.CustomAbort(http.StatusInternalServerError, "unauthorized,operation(pushing) is not authorized")
	}

	if destProject == nil {
		ra.CustomAbort(http.StatusNotFound, fmt.Sprintf("project %s not found", destProjectName))
	}

	if destProject.Public == 0 {
		userID := ra.ValidateUser()
		if !hasProjectDevRole(userID, destProject.ProjectID) {
			ra.CustomAbort(http.StatusForbidden, "unauthorized,operation(pushing) is not authorized")
		}
	}
	//check quota

	usedNum, usedSize, quotaNum, quotaSize, err := dao.GetInforByProject(destProjectName)
	if err != nil {
		log.Errorf("Error happens get the  quota information of the project:%v ", err)
	}
	if (usedNum >= quotaNum) || (usedSize >= quotaSize*token.GetThreshold()) {
		log.Errorf("project %s has not enough quota %v", destProjectName)
		ra.CustomAbort(http.StatusForbidden, "unauthorized,operation(pushing) is not authorized，the project quota reaches the upper limit")
	}

	//new rource and destination client
	srcRc, err := ra.initRepositoryClient(srcRepoName)
	if err != nil {
		log.Errorf("error occurred while initializing repository client for %s: %v", srcRepoName, err)
		ra.CustomAbort(http.StatusInternalServerError, "internal error")
	}
	destRc, err := ra.initRepositoryClient(destRepoName)
	if err != nil {
		log.Errorf("error occurred while initializing repository client for %s: %v", destRepoName, err)
		ra.CustomAbort(http.StatusInternalServerError, "internal error")
	}

	//get manifest
	mediaTypes := []string{}
	mediaTypes = append(mediaTypes, schema2.MediaTypeManifest)

	//pull manifest from registry
	_, mediaType, payload, err := srcRc.PullManifest(srcTag, mediaTypes)
	if err != nil {
		log.Errorf("error occurred while getting manifest of %s:%s: %v", srcRepoName, srcTag, err)
		ra.CustomAbort(http.StatusInternalServerError, "internal error")
	}

	//converts []byte to be distribution.Manifest
	manifest, _, err := registry.UnMarshal(mediaType, payload)
	if err != nil {
		log.Errorf("an error occurred while parsing manifest of %s:%s: %v", srcRepoName, srcTag, err)
		ra.CustomAbort(http.StatusInternalServerError, "internal error")
	}
	//pull and push blobs
	go func() {
		var srcBlobs []string
		var blobs []string
		for _, discriptor := range manifest.References() {
			srcBlobs = append(srcBlobs, discriptor.Digest.String())
		}

		// config is also need to be transferred if the schema of manifest is v2
		manifest2, ok := manifest.(*schema2.DeserializedManifest)
		if ok {
			srcBlobs = append(srcBlobs, manifest2.Target().Digest.String())
		}

		log.Infof("all blobs of %s:%s :%v", srcRepoName, srcTag, srcBlobs)

		//var blobsExistence map[string]bool
		for _, blob := range srcBlobs {
			exist, err := destRc.BlobExist(blob)
			if err != nil {
				log.Errorf("an error occurred while checking existing of blobs %s:%s: %v", destRepoName, destTag, err)
			}
			if !exist {
				blobs = append(blobs, blob)
			} else {
				log.Infof("blob %s of %s:%s already exists ", blob, destRepoName, destTag)
			}
		}
		//push blobs
		for _, blob := range blobs {
			log.Infof("copying blob  %s of %s:%s  ", blob, destRepoName, destTag)
			size, data, err := srcRc.PullBlob(blob)
			if err != nil {
				log.Errorf("an error occurred while pulling blob %s of %s:%s :%v", blob, srcRepoName, srcTag, err)
				return
			}
			if data != nil {
				defer data.Close()
			}
			if err = destRc.PushBlob(blob, size, data); err != nil {
				log.Errorf("an error occurred while pushing blob %s of %s:%s  : %v", blob, destRepoName, destTag, err)
				return
			}
		}
		log.Infof("all blobs of %s:%s has been pushed ", destRepoName, destTag)
		mediaType, data, err := manifest.Payload()
		if err != nil {
			log.Errorf("an error occurred while getting payload of manifest for %s:%s : %v", destRepoName, destTag, err)
			return
		}
		if _, err = destRc.PushManifest(destTag, mediaType, data); err != nil {
			log.Infof("an error occurred while pushing manifest of %s:%s : %v", destRepoName, destTag, err)
			return
		}
		log.Infof("manifest of %s:%s has been pushed ", destRepoName, destTag)
	}()
	//go TriggerReplicationByRepositorySync(destRepoName, []string{destTag}, models.RepOpTransfer)
}


//CopyPro handles request Post /api/project/copyProjects
type copyProject struct {
	SrcProNames  []string `json:"src_pro_names"`
	Prefix      string `json:"prefix"`
	Suffix      string `json:"suffix"`
}

//CopyPro handles request Post /api/project/copyProjects
func (pca *RepositoryAPI) CopyPro() {
	userID := pca.ValidateUser()

	var cp copyProject
	pca.DecodeJSONReq(&cp)

	var newPro string

	spn := cp.SrcProNames
	pf := cp.Prefix
	sf := cp.Suffix

	// check value
	if spn == nil || len(spn) <= 0 {
		pca.CustomAbort(http.StatusBadRequest, "src_pro_names is nil")
	}

	// loop projects
	for _, project := range spn {
		if len(pf) > 0 {
			newPro = pf + project
		}

		if len(sf) > 0 {
			newPro = project + sf
		}

		// if not exist?(create)
		exist, err := dao.ProjectExists(newPro)
		if err != nil {
			log.Errorf("Error happened checking project existence in db, error: %v, project name: %s", err, newPro)
		}

		if !exist {
			project := models.Project{OwnerID: userID, Name: newPro, CreationTime: time.Now(), Public: 1}
			projectID, err := dao.AddProject(project)
			if err != nil {
				log.Errorf("Failed to add project, error: %v", err)
				dup, _ := regexp.MatchString(dupProjectPattern, err.Error())
				if dup {
					pca.RenderError(http.StatusConflict, "")
				} else {
					pca.RenderError(http.StatusInternalServerError, "Failed to add project")
				}
				return
			}
			pca.Redirect(http.StatusCreated, strconv.FormatInt(projectID, 10))
			//@lili start
			//get the quota from request
			projectQuotaNum := 10000
			projectQuotaSize := float64(1000)
			//need update the initial num
			quotaid, err := dao.AddProjectQuota(projectID, newPro, projectQuotaNum, 0, projectQuotaSize, 0.0)
			if err != nil {
				log.Errorf("Error happens insert project_quota: %v", err)
			}
			log.Debugf("The result of insert project_quota: %d", quotaid)
		}
		// list srcPro repo:tag
		repositories, err := getReposByProject(project)
		if err != nil {
			log.Errorf("failed to get repository: %v", err)
			pca.CustomAbort(http.StatusInternalServerError, "")
		}

		if repositories != nil && len(repositories) > 0 {
			for _, repoName := range repositories {
				rc, err := pca.initRepositoryClient(repoName)
				if err != nil {
					log.Errorf("error occurred while initializing repository client for %s: %v", repoName, err)
					pca.CustomAbort(http.StatusInternalServerError, "internal error")
				}

				ts, err := rc.ListTag()
				for _, tag := range ts {
					var image_name = repoName + ":" + tag


				}

			}
		}

	}


	// copy image (Similar to CopyRepo)
}
