package api

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vmware/harbor/dao"
	"github.com/vmware/harbor/models"
	"github.com/vmware/harbor/service/cache"
	"github.com/vmware/harbor/utils/log"
	"github.com/vmware/harbor/utils/registry"
	registry_error "github.com/vmware/harbor/utils/registry/error"
)

type ImageAPI struct {
	BaseAPI
	uID int
}

//Prepare validates the URL and the user
func (ia *ImageAPI) Prepare() {
	ia.uID = ia.ValidateUser()
}

type ImageList struct {
	UserID         int       `json:"user_id"`
	ProjectID      int64     `json:"project_id"`
	RepositoryName string    `json:"repo_name"`
	RepositoryTag  string    `json:"repo_tag"`
	UpdateTime     time.Time `json:"update_time“`
	Overdue        float64   `json:"overdue"`
}

type imageInput struct {
	ImageInfo []imageInfo `json:"image_info"`
}

type imageInfo struct {
	UserID   int    `json:"user_id"`
	ProID    int64  `json:"project_id"`
	RepoName string `json:"repo_name"`
	RepoTag  string `json:"repo_tag"`
}

func (ia *ImageAPI) GetImageList() {
	imageList := []ImageList{}
	iml := ImageList{}
	var image_result []models.AccessLog
	var err error
	isAdmin := false
	t, _ := ia.GetInt("time")
	if t == 0 {
		ia.CustomAbort(http.StatusBadRequest, "time is null")
	}

	ia.uID = ia.ValidateUser()
	isAdmin, err = dao.IsAdminRole(ia.uID)
	if err != nil {
		log.Errorf("Error occured in check admin, error: %v", err)
		ia.CustomAbort(http.StatusInternalServerError, "Internal error.")
	}

	if isAdmin { //To determine the admin use
		userNum, err := dao.GetAllUserID()
		if err != nil {
			log.Errorf("error happens when get user from sql: %v", err)
			ia.CustomAbort(http.StatusInternalServerError, "")
		}
		for _, u := range userNum {
			allUserTagNum := GetTagByUser(u, ia)
			for _, userTag := range allUserTagNum {
				r := strings.Split(userTag, ":")[0]
				tag := strings.Split(userTag, ":")[1]
				ir, err := dao.GetUserImageList(r, tag, t, u)
				if err != nil {
					log.Errorf("error happens when get Imagelist from sql: %v", err)
					ia.CustomAbort(http.StatusInternalServerError, "")
				}
				image_result = ia.merge(image_result, ir)
			}
		}

	} else {
		adminDeTime := "admin_detectTime"
		t, err := dao.GetAdminTime(adminDeTime)
		if err != nil {
			log.Errorf("error happens when get Imagelist from sql: %v", err)
			ia.CustomAbort(http.StatusInternalServerError, "Error happens when get admin_detectetime from sql")
		}
		if t == 0 {
			log.Errorf("error happens when get no time: %v", err)
			ia.CustomAbort(http.StatusBadRequest, "detection time is null")
		}

		allTagNum := GetTagByUser(ia.uID, ia)
		for _, userTag := range allTagNum {
			r := strings.Split(userTag, ":")[0]
			tag := strings.Split(userTag, ":")[1]
			ir, _ := dao.GetUserImageList(r, tag, t, ia.uID)
			image_result = ia.merge(image_result, ir)
		}
	}

	for _, ir := range image_result {
		if iml.UserID != 0 { //To determine the first cycle
			imageList = append(imageList, iml)
			iml = cleanIML()
		}

		iml.UserID = ir.UserID
		iml.ProjectID = ir.ProjectID
		iml.RepositoryName = ir.RepoName
		iml.RepositoryTag = ir.RepoTag
		iml.UpdateTime = ir.LatestTime
		iml.Overdue = ir.Overdue
	}

	imageList = append(imageList, iml)
	ia.Data["json"] = imageList
	ia.ServeJSON()
	return
}

func cleanIML() (iml ImageList) {
	return ImageList{}
}

func (ia *ImageAPI) initRepositoryClient(repoName string) (r *registry.Repository, err error) {
	endpoint := os.Getenv("REGISTRY_URL")

	username, password, ok := ia.Ctx.Request.BasicAuth()
	if ok {
		return newRepositoryClient(endpoint, getIsInsecure(), username, password,
			repoName, "repository", repoName, "pull", "push", "*")
	}

	username, err = ia.getUsername()
	if err != nil {
		return nil, err
	}

	return cache.NewRepositoryClient(endpoint, getIsInsecure(), username, repoName,
		"repository", repoName, "pull", "push", "*")
}

func (ia *ImageAPI) getUsername() (string, error) {
	// get username from session
	sessionUsername := ia.GetSession("username")
	if sessionUsername != nil {
		username, ok := sessionUsername.(string)
		if ok {
			return username, nil
		}
	}

	// if username does not exist in session, try to get userId from sessiion
	// and then get username from DB according to the userId
	sessionUserID := ia.GetSession("userId")
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

func GetTagByRepo(repo string, ia *ImageAPI) []string {
	rc, err := ia.initRepositoryClient(repo)
	if err != nil {
		log.Errorf("error occurred while initializing repository client for %s: %v", repo, err)
		ia.CustomAbort(http.StatusInternalServerError, "internal error")
	}

	tags := []string{}
	repoTags := []string{}

	ts, err := rc.ListTag()
	if err != nil {
		regErr, ok := err.(*registry_error.Error)
		if !ok {
			log.Errorf("error occurred while listing tags of %s: %v", repo, err)
			ia.CustomAbort(http.StatusInternalServerError, "internal error")
		}
		// TODO remove the logic if the bug of registry is fixed
		// It's a workaround for a bug of registry: when listing tags of
		// a repository which is being pushed, a "NAME_UNKNOWN" error will
		// been returned, while the catalog API can list this repository.
		if regErr.StatusCode != http.StatusNotFound {
			ia.CustomAbort(regErr.StatusCode, regErr.Detail)
		}
	}

	tags = append(tags, ts...)

	for _, tag := range tags {
		tag = repo + ":" + tag
		repoTags = append(repoTags, tag)
	}

	return repoTags
}

func GetTagByProject(projectName string, ia *ImageAPI) []string {
	projectSum := []string{}
	projectID, err := dao.GetProjectID(projectName)
	if err != nil {
		log.Errorf("failed to get projectID by projectName erro: %v", err)
	}

	repoList, err := dao.GetByProjectId(projectID)
	if err != nil {
		log.Errorf("failed to get repoList by projectID erro: %v", err)
	}

	for _, repo := range repoList {
		repoSum := GetTagByRepo(repo.Name, ia)
		projectSum = merge(projectSum, repoSum)
	}
	return projectSum

}

func GetTagByUser(userID int, ia *ImageAPI) []string {
	userSum := []string{}

	projectList, err := dao.GetProjectByOwner(userID)
	if err != nil {
		log.Errorf("failed to get repoList by projectID erro: %v", err)
	}

	for _, project := range projectList {
		repoSum := GetTagByProject(project.Name, ia)
		userSum = merge(userSum, repoSum)
	}

	return userSum
}

func (ia *ImageAPI) merge(a []models.AccessLog, b []models.AccessLog) []models.AccessLog {
	c := make([]models.AccessLog, len(a)+len(b))
	copy(c, a)
	copy(c[len(a):], b)
	return c
}

//ExtendImage handles request GET /api/image/extendImage
func (ia *ImageAPI) ExtendImage() {
	var input imageInput
	ia.DecodeJSONReq(&input)
	if len(input.ImageInfo) == 0 {
		log.Error("error happens ip or runtime result is nil")
		ia.CustomAbort(http.StatusBadRequest, "input is null")
	}

	var info imageInfo
	for _, result := range input.ImageInfo {
		info.UserID = result.UserID
		info.ProID = result.ProID
		info.RepoName = result.RepoName
		info.RepoTag = result.RepoTag

		if err := dao.AddAccessLog(models.AccessLog{
			UserID:    info.UserID,
			ProjectID: info.ProID,
			RepoName:  info.RepoName,
			RepoTag:   info.RepoTag,
			Operation: "extend",
		}); err != nil {
			log.Errorf("failed to add access log: %v", err)
			ia.CustomAbort(http.StatusInternalServerError, "Error when extend image.")
		}
	}
}
