package api

import (
	"bufio"
	//	"io/ioutil"
	"os"

	"strings"
	"time"

	"github.com/docker/distribution"
	"github.com/vmware/harbor/dao"
	"github.com/vmware/harbor/service/scan"
	"github.com/vmware/harbor/utils/log"
	"github.com/vmware/harbor/utils/registry"
)

//const logFilePath = `/var/log/image-log/`
const MediaTypeManifest = "application/vnd.docker.distribution.manifest.v2+json"

type ScanImagesAPI struct {
	BaseAPI
}

type scanMassage struct {
	Username    string   `json:"user_name"`
	Projectname string   `json:"project_name"`
	Imagenames  []string `json:"image_names"` //projectname/imagename:tag
}

func (si *ScanImagesAPI) ScanImages() {

	userID := si.ValidateUser()
	userName, err := dao.GetUserName(userID)
	if err != nil {
		log.Errorf("failed to get user: %v", err)
	}

	var sm scanMassage

	si.DecodeJSONReq(&sm)

	if len(sm.Imagenames) > 0 {
		for _, imageName := range sm.Imagenames {

			scanImageAndUpdate(userName, imageName)

		}
	} else if sm.Projectname != "" {
		images, err := dao.GetClairRepoNameByProjectName(sm.Projectname)
		if err != nil {
			log.Errorf("Error happens get repo_name by project name: %v", err)
		}
		for _, image := range images {
			scanImageAndUpdate(userName, image.RepoName)
		}
	} else if sm.Username != "" {
		images, err := dao.GetClairRepoNameByUserName(sm.Username)
		if err != nil {
			log.Errorf("Error happens get repo_name by user name: %v", err)
		}
		for _, image := range images {
			scanImageAndUpdate(userName, image.RepoName)
		}
	} else {
		images, err := dao.GetClairAllRepoName()
		if err != nil {
			log.Errorf("Error happens get all repo_name: %v", err)
		}
		for _, image := range images {
			scanImageAndUpdate(userName, image.RepoName)
		}
	}
}

func (si *ScanImagesAPI) InitCVEStatus() {
	err := dao.InitCVE()
	if err != nil {
		log.Errorf("Error happens init CVE status: %v", err)
	}
}

func scanImageAndUpdate(userName string, imageName string) {
	image := strings.Split(imageName, ":")

	if len(image) != 2 {
		log.Errorf("Data format error")
		return
	}
	repository := image[0]
	tag := image[1]

	var otherStatus int
	ch := make(chan int, 1)
	go func() {
		ch <- 1
		image_size, content, err := scanImage(userName, MediaTypeManifest, repository, tag)
		if err != nil {
			log.Errorf("Failed to scan image: %v", err)
		}
		if image_size == 0 {
			log.Errorf("Failed to get size of image")
		}

		hn, on := CVECount(content)

		if hn > 0 {
			otherStatus = 5
		} else {
			otherStatus = OtherStats(content)
		}
		if err != nil {
			log.Errorf("Failed to check image size: %v", err)
			err1 := dao.UpdateClairResult(imageName, content, hn, on, 1, otherStatus)
			if err1 != nil {
				log.Errorf("Error happens update clair_result: %v", err1)
			}
		} else {
			err1 := dao.UpdateClairResultSize(imageName, content, hn, on, 1, otherStatus, image_size)
			if err1 != nil {
				log.Errorf("Error happens update clair_result: %v", err1)
			}
		}
	}()
	select {
	case <-ch:
	case <-time.After(10 * time.Second):
		log.Errorf("can not start clair")
	}
}

func OtherStats(content string) int {
	if strings.Contains(content, "the image isn't supported by Clair") {
		return 1
	} else if strings.Contains(content, "Success!") {
		return 2
	} else if strings.Contains(content, "CVE-") || strings.Contains(content, "RHSA-") {
		return 4
	}
	return 3
}

//TODO need to rebuild
func CVECount(content string) (int, int) {
	highNum := 0
	otheNum := 0
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if (strings.HasPrefix(line, "CVE-") || strings.HasPrefix(line, "RHSA-"))&& strings.Contains(line, "(") {
			if strings.Contains(line, "Low") {
				otheNum++
			} else if strings.Contains(line, "Negligible") {
				otheNum++
			} else if strings.Contains(line, "Medium") {
				otheNum++
			} else if strings.Contains(line, "High") {
				highNum++
			} else if strings.Contains(line, "Unknown") {
				otheNum++
			}
		}
	}
	return highNum, otheNum
}
func scanImage(user string, mediaType string, repository string, tag string) (float64, string, error) {
	log.Info("%%%%%%%%%%%%%%%%%%%%%%%%%%%%% get manifest start")
	manifest, err := getManifest(user, mediaType, repository, tag)
	if err != nil {
		log.Errorf("error occurred while getting manifest for %s:%s : %v", repository, tag, err)
		return 0, "", err
	}
	log.Info("%%%%%%%%%%%%%%%%%%%%%%%%%%%%% get manifest end")
	image_size, result, err := scan.ScanManifest(user, manifest, repository, tag)
	return image_size, result, err
}
func getManifest(username string, currentmediaType string, repoName string, tag string) (distribution.Manifest, error) {
	//get the registry url from env .  mediatype (v1 or v2)

	endpoint := os.Getenv("REGISTRY_URL")
	rc, err := scan.CreateRepositoryClient(endpoint, GetIsInsecure(), username, repoName)
	if err != nil {
		log.Errorf("error occurred while creating repository client for %s: %v", repoName, err)
		return nil, err
	}

	mediaTypes := []string{}
	mediaTypes = append(mediaTypes, currentmediaType)

	//pull manifest from registry
	_, mediaType, payload, err := rc.PullManifest(tag, mediaTypes)
	if err != nil {
		log.Errorf("error occurred while getting manifest of %s:%s: %v", repoName, tag, err)
		return nil, err
	}

	//converts []byte to be distribution.Manifest
	manifest, _, err := registry.UnMarshal(mediaType, payload)
	if err != nil {
		log.Errorf("an error occurred while parsing manifest of %s:%s: %v", repoName, tag, err)
		return nil, err
	}
	log.Info("toplayer" + manifest.References()[0].Digest.String())
	return manifest, nil
}
