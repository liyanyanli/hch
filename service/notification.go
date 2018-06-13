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

package service

import (
	"encoding/json"
	"regexp"
	"strings"
	//"time"

	"github.com/docker/distribution"
	"github.com/vmware/harbor/api"
	"github.com/vmware/harbor/dao"
	"github.com/vmware/harbor/models"
	"github.com/vmware/harbor/service/cache"
	"github.com/vmware/harbor/service/scan"
	"github.com/vmware/harbor/utils"
	"github.com/vmware/harbor/utils/log"
	"github.com/vmware/harbor/utils/registry"

	"os"
	//	"os/exec"
	"path/filepath"

	"github.com/astaxie/beego"
	"time"
)

// NotificationHandler handles request on /service/notifications/, which listens to registry's events.
type NotificationHandler struct {
	beego.Controller
}

const manifestPattern = `^application/vnd.docker.distribution.manifest.v\d\+json`

// Post handles POST request, and records audit log or refreshes cache based on event.
func (n *NotificationHandler) Post() {

	log.Debug("Recive Notification from registry!!!!")
	var notification models.Notification

	//	var wg sync.WaitGroup
	err := json.Unmarshal(n.Ctx.Input.CopyBody(1<<32), &notification)

	if err != nil {
		log.Errorf("failed to decode notification: %v", err)
		return
	}

	events, err := filterEvents(&notification)
	if err != nil {
		log.Errorf("failed to filter events: %v", err)
		return
	}

	for _, event := range events {
		repository := event.Target.Repository
		project, _ := utils.ParseRepository(repository)
		tag := event.Target.Tag
		action := event.Action
		user := event.Actor.Name
		mediaType := event.Target.MediaType

		if len(user) == 0 {
			user = "anonymous"
		}

		go func() {
			//insert access log after image scan is finished
			time.Sleep(1500 * time.Millisecond)
			if err := dao.AccessLog(user, project, repository, tag, action); err != nil {
				log.Errorf("failed to add access log: %v", err)
			}
		}()

		//log.Debugf("***********Action: ", action)

		if action == "push" {
			//log.Debug("***********Recive push!!!!*****************")
			log.Infof("mediaType :%s", mediaType)
			exist := dao.RepositoryExists(repository)
			if !exist {
				log.Debugf("Add repository %s into DB.", repository)
				repoRecord := models.RepoRecord{Name: repository, OwnerName: user, ProjectName: project}
				if err := dao.AddRepository(repoRecord); err != nil {
					log.Errorf("Error happens when adding repository: %v", err)
				}
				if err := cache.RefreshCatalogCache(); err != nil {
					log.Errorf("Failed to refresh cache: %v", err)
				}
			}

			go api.TriggerReplicationByRepositorySync(repository, []string{tag}, models.RepOpTransfer)

			//check img
			go func() {
				imgName := repository + ":" + tag
				err := addQuotaNum(imgName, project)
				if err != nil {
					log.Errorf("Failed to update the used number of project: %v", err)
				}
				log.Info("mediatype:  %s", mediaType)
				image_size, result, err := scanImage(user, mediaType, repository, tag)
				if err != nil {
					log.Errorf("Failed to scan image: %v", err)
				}
				if image_size == 0 {
					log.Errorf("Failed to get size of image")
				}

				fileName := repository + "_" + tag
				mail, err := dao.GetEmailByUsername(user)
				if err != nil {
					log.Errorf("Failed to select email: %v", err)
				}

				err = CheckImageSafety(mail, result, imgName, user, project)
				if err == nil {
					// When pull and analyze image finished, the pull count -1.
					errCount := dao.DecreasePullCount(repository)
					if errCount != nil {
						log.Errorf("Error happens decrease pull count: %v", errCount)
					}
				} else {
					log.Errorf("Error happens check image with clair: %v", err)
				}

				err = CheckImageSize(fileName, imgName, user, project, image_size)
				if err != nil {
					log.Errorf("Error happens update the size of image: %v", err)
				}

			}()

		}
		if action == "pull" {
			//go func() {
			log.Debugf("Increase the repository %s pull count.", repository)
			if err := dao.IncreasePullCount(repository); err != nil {
				log.Errorf("Error happens when increasing pull count: %v", repository)
			}
			//}()
		}
	}
}

func filterEvents(notification *models.Notification) ([]*models.Event, error) {
	events := []*models.Event{}

	for _, event := range notification.Events {
		log.Debugf("receive an event: ID-%s, target-%s:%s, digest-%s, action-%s", event.ID, event.Target.Repository, event.Target.Tag,
			event.Target.Digest, event.Action)

		isManifest, err := regexp.MatchString(manifestPattern, event.Target.MediaType)
		if err != nil {
			log.Errorf("failed to match the media type against pattern: %v", err)
			continue
		}

		if !isManifest {
			continue
		}
		//log.Infof("LKLKLKLKLKLKLKLKLK" + event.Request.UserAgent)
		//pull and push manifest by docker-client
		if strings.HasPrefix(event.Request.UserAgent, "docker") && (event.Action == "pull" || event.Action == "push") {
			events = append(events, &event)
			log.Debugf("add event to collect: %s", event.ID)
			continue
		}

		//push manifest by docker-client or job-service
		if strings.ToLower(strings.TrimSpace(event.Request.UserAgent)) == "harbor-registry-client" && event.Action == "push" {
			events = append(events, &event)
			log.Debugf("add event to collect: %s", event.ID)
			continue
		}
		//pull and push manifest by Go-http-client
		if strings.HasPrefix(event.Request.UserAgent, "Go-http-client") && (event.Action == "pull" || event.Action == "push") {
			events = append(events, &event)
			log.Debugf("add event to collect: %s", event.ID)
			continue
		}

	}

	return events, nil
}

// Render returns nil as it won't render any template.
func (n *NotificationHandler) Render() error {
	return nil
}

// get the filePath.
func substr(s string, pos, length int) string {
	runes := []rune(s)
	l := pos + length
	if l > len(runes) {
		l = len(runes)
	}
	return string(runes[pos:l])
}

func getParentDirectory(dirctory string) string {
	return substr(dirctory, 0, strings.LastIndex(dirctory, "/"))
}

func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Errorf("failed to get the current direcroty: %s", err)
	}
	return strings.Replace(dir, "\\", "/", -1)
}

func scanImage(user string, mediaType string, repository string, tag string) (float64, string, error) {
	//log.Info("%%%%%%%%%%%%%%%%%%%%%%%%%%%%% get manifest start")
	manifest, err := getManifest(user, mediaType, repository, tag)
	if err != nil {
		if strings.Contains(err.Error(), "MANIFEST_UNKNOWN") {
			//registry tag link may not be created when received this notification, retry 1 second later
			time.Sleep(1 * time.Second)
			manifestRes, err := getManifest(user, mediaType, repository, tag)
			if err != nil {
				log.Errorf("error occurred while retry getting manifest for %s:%s : %v", repository, tag, err)
				return 0, "", err
			}
			manifest = manifestRes;
		} else {
			log.Errorf("error occurred while getting manifest for %s:%s : %v", repository, tag, err)
			return 0, "", err
		}
	}
	//log.Info("%%%%%%%%%%%%%%%%%%%%%%%%%%%%% get manifest end")
	image_size, result, err := scan.ScanManifest(user, manifest, repository, tag)
	return image_size, result, err
}

func getManifest(username string, currentmediaType string, repoName string, tag string) (distribution.Manifest, error) {
	//get the registry url from env .  mediatype (v1 or v2)

	endpoint := os.Getenv("REGISTRY_URL")
	rc, err := scan.CreateRepositoryClient(endpoint, api.GetIsInsecure(), username, repoName)
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
