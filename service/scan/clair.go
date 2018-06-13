package scan

import (
	"bytes"
	"encoding/json"
	"errors"

	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/docker/distribution"
	"github.com/vmware/harbor/service/cache"
	"github.com/vmware/harbor/service/token"
	"github.com/vmware/harbor/utils/log"
	"github.com/vmware/harbor/utils/registry"
)

const (
	tokenService = "token-service"
)
const EMPTY_LAYER_BLOB_SUM = "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"

//const PATH = "/var/log/image-log/"

var currentregistry string
var clairURL string

// FsLayer represents a layer in docker image
type FsLayer struct {
	BlobSum string
}
type Image struct {
	Registry string
	RepoName string
	Tag      string
	FsLayers []FsLayer
	Token    string
	user     string
}

type layer struct {
	Name             string    `json:"Name,omitempty"`
	NamespaceName    string    `json:"NamespaceName,omitempty"`
	Path             string    `json:"Path,omitempty"`
	Headers          headers   `json:"Headers,omitempty"`
	ParentName       string    `json:"ParentName,omitempty"`
	Format           string    `json:"Format,omitempty"`
	IndexedByVersion int       `json:"IndexedByVersion,omitempty"`
	Features         []feature `json:"Features,omitempty"`
}

// Vulnerability represents vulnerability entity returned by Clair
type Vulnerability struct {
	Name          string                 `json:"Name,omitempty"`
	NamespaceName string                 `json:"NamespaceName,omitempty"`
	Description   string                 `json:"Description,omitempty"`
	Link          string                 `json:"Link,omitempty"`
	Severity      string                 `json:"Severity,omitempty"`
	Metadata      map[string]interface{} `json:"Metadata,omitempty"`
	FixedBy       string                 `json:"FixedBy,omitempty"`
	FixedIn       []feature              `json:"FixedIn,omitempty"`
}

type feature struct {
	Name            string          `json:"Name,omitempty"`
	NamespaceName   string          `json:"NamespaceName,omitempty"`
	Version         string          `json:"Version,omitempty"`
	Vulnerabilities []Vulnerability `json:"Vulnerabilities"`
	AddedBy         string          `json:"AddedBy,omitempty"`
}

type Clair struct {
	URL string
}

type headers struct {
	Authorization string
}

type clairError struct {
	Message string `json:"Layer"`
}

type layerEnvelope struct {
	Layer *layer      `json:"Layer,omitempty"`
	Error *clairError `json:"Error,omitempty"`
}

// @ vulnerability information
type vulnerabilityInfo struct {
	vulnerability Vulnerability
	features      feature
	severity      Priority
}

type sorter struct {
	vulnerabilities []vulnerabilityInfo
	by              func(v1, v2 vulnerabilityInfo) bool
}

func (s *sorter) Len() int {
	return len(s.vulnerabilities)
}

func (s *sorter) Swap(i, j int) {
	s.vulnerabilities[i], s.vulnerabilities[j] = s.vulnerabilities[j], s.vulnerabilities[i]
}

func (s *sorter) Less(i, j int) bool {
	return s.by(s.vulnerabilities[i], s.vulnerabilities[j])
}

type By func(v1, v2 vulnerabilityInfo) bool

func (by By) Sort(vulnerabilities []vulnerabilityInfo) {
	ps := &sorter{
		vulnerabilities: vulnerabilities,
		by:              by,
	}
	sort.Sort(ps)
}

//  pullManifest pull the image's manifest and return it

//  create RepositoryClient which along with token
func CreateRepositoryClient(endpoint string, insecure bool, username string, repoName string) (r *registry.Repository, err error) {
	return cache.NewRepositoryClient(endpoint, insecure, username, repoName,
		"repository", repoName, "pull", "push", "*")
}

//  newImage create an image,analse manifest to get the layers digest
func newImage(registry string, repoName string, tag string, user string, manifest distribution.Manifest) (*Image, error) {
	var fslayers []FsLayer
	if len(manifest.References()) == 0 {
		log.Info("manifest length is 0")
	}
	log.Info("manifest length is ", len(manifest.References()))
	for i := 0; i < len(manifest.References()); i++ {
		log.Info("degest:", i, manifest.References()[i].Digest.String())
		fs := FsLayer{}
		fs.BlobSum = manifest.References()[i].Digest.String()
		fslayers = append(fslayers, fs)

		//fslayers[i].BlobSum = manifest.References()[i].Digest.String()
		log.Info("fslayer:", i, fslayers[i].BlobSum)
	}

	fslayers = filterEmptyLayers(fslayers)
	scope := "repository:" + repoName + ":pull"
	access := token.GetResourceActions([]string{scope})
	token, _, _, err := token.MakeToken(user, tokenService, access)
	if err != nil {
		log.Errorf("error happens when creating token, error: %v", err)
		return nil, err
	}
	log.Info("token:", token)
	return &Image{
		Registry: registry,
		RepoName: repoName,
		Tag:      tag,
		FsLayers: fslayers,
		Token:    token,
		user:     user,
	}, nil
}

//  Analyse sent each layer from Docker image to Clair and returns a list of found vulnerabilities
func (c *Clair) analyse(image *Image) (string, error) {
	//check the length of layers
	layerLength := len(image.FsLayers)
	if layerLength == 0 {
		log.Errorf("No need to analyse image %s/%s:%s as there is no non-emtpy layer")
		return "", nil
	}
	//	var vs []Vulnerability
	for i := 0; i <= layerLength-1; i++ {

		layer := newLayer(image, i)
		err := c.pushLayer(layer)
		if err != nil {
			log.Errorf("Push layer %d failed: %s", i, err.Error())
			continue
		}
	}
	result, err := c.analyzeLayer(image.FsLayers[layerLength-1])
	return result, err
}

//  create new layser,"layer" is the describe of current layer
//( eg name:      sha256:193d4969ca79a1eae056433a65e49d650e697d55f280d568f213d0eccc23ac50)
func newLayer(image *Image, index int) *layer {
	var parentName string
	//if index < len(image.FsLayers)-1 {
	if index > 0 {
		parentName = image.FsLayers[index-1].BlobSum
		log.Info("parentName:" + parentName)
	}
	log.Info("path:" + strings.Join([]string{image.Registry, image.RepoName, "blobs", image.FsLayers[index].BlobSum}, "/"))
	return &layer{
		Name:       image.FsLayers[index].BlobSum,
		Path:       strings.Join([]string{image.Registry, image.RepoName, "blobs", image.FsLayers[index].BlobSum}, "/"),
		ParentName: parentName,
		Format:     "Docker",
		Headers:    headers{"Bearer " + image.Token},
	}
}

// @lili push the layer's information to clair,clair pull the layer from registry according to the information
func (c *Clair) pushLayer(layer *layer) error {
	envelope := layerEnvelope{Layer: layer}
	reqBody, err := json.Marshal(envelope)
	if err != nil {
		log.Errorf("can't serialze push request: %s", err)
		return err
	}
	url := c.URL + "/layers"
	//log.Info("********************: ", url)
	//log.Errorf("********************: %s", reqBody)
	//log.Info("********************: ", layer.Name)
	//log.Info("********************: ", layer.ParentName)
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))

	if err != nil {
		log.Errorf("Can't create a push request: %s", err)
		return err

	}
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: time.Minute}).Do(request)
	if err != nil {
		log.Errorf("Can't push layer to Clair: %s", err)
		return err

	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Can't read clair response : %s", err)
		return err
	}
	log.Info("url: responsecode:", url, response.StatusCode)
	if response.StatusCode != http.StatusCreated {
		var errResp string
		err = json.Unmarshal(body, &errResp)
		if err != nil {
			log.Errorf("Can't even read an error message: %s", err)
			return err
		}
		log.Errorf("Push error %d: %s", response.StatusCode, string(body))
		return err
	}
	return nil
}

// @lili get the result of image by sending the top layer to clair
func (c *Clair) analyzeLayer(layer FsLayer) (string, error) {
	var result string
	var midResult string
	url := c.URL + "/layers/" + layer.BlobSum + "?vulnerabilities"
	//log.Info("~~~~~~~~~~~url:" + url)
	response, err := http.Get(url)
	if err != nil {
		log.Errorf("send http request error ")
		return "", err
	}
	//log.Info("~~~~~~~~~~~~~result responsecode:", response.StatusCode)
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		log.Errorf("Analyze error " + string(body))
		return "", errors.New("Analyze error " + string(body))

	}
	var envelope layerEnvelope
	if err = json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		log.Errorf("decode body error ")
		return "", err
	}
	currentlayer := *envelope.Layer
	if len(currentlayer.Features) == 0 {
		//log.Infof("No features have been detected in the image. This usually means that the image isn't supported by Clair.\n")
		result = "NOTE:" + " No features have been detected in the image. This usually means that the image isn't supported by Clair.\n"
		return result, nil
	}
	isSafe := true
	hasVisibleVulnerabilities := false
	minSeverity := Priority(*flagMinimumSeverity)

	var vulnerabilities = make([]vulnerabilityInfo, 0)
	for _, feature := range currentlayer.Features {
		if len(feature.Vulnerabilities) > 0 {
			for _, vulnerability := range feature.Vulnerabilities {
				severity := Priority(vulnerability.Severity)
				isSafe = false

				if minSeverity.Compare(severity) > 0 {
					continue
				}

				hasVisibleVulnerabilities = true
				vulnerabilities = append(vulnerabilities, vulnerabilityInfo{vulnerability, feature, severity})
			}
		}
	}
	// Sort vulnerabilitiy by severity.
	priority := func(v1, v2 vulnerabilityInfo) bool {
		return v1.severity.Compare(v2.severity) >= 0
	}

	By(priority).Sort(vulnerabilities)
	//err=writevul(repository string, tag string，vulnerabilities []vulnerabilityInfo)
	/*
		fl, err := os.OpenFile(file, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.Errorf("can not create dir,error message%s : %s", err)
			return err
		}
	*/
	for _, vulnerabilityInfo := range vulnerabilities {
		//需要写入文件
		vulnerability := vulnerabilityInfo.vulnerability
		feature := vulnerabilityInfo.features
		severity := vulnerabilityInfo.severity

		//log.Infof("%s (%s)\n", vulnerability.Name, severity)

		//fl.WriteString(vulnerability.Name + " (" + (string)(severity) + ")\n")
		midResult = vulnerability.Name + " (" + (string)(severity) + ")\n"
		result += midResult
		if vulnerability.Description != "" {
			//log.Infof("%s\n\n", vulnerability.Description)
			midResult = "\t" + vulnerability.Description + "\n\n"
			result += midResult
			//			fl.WriteString("\t" + vulnerability.Description + "\n\n")
		}

		//log.Infof("\tPackage:       %s @ %s\n", feature.Name, feature.Version)
		//fl.WriteString("\tPackage:       " + feature.Name + " @ " + feature.Version + "\n")
		midResult = "\tPackage:       " + feature.Name + " @ " + feature.Version + "\n"
		result += midResult

		if vulnerability.FixedBy != "" {
			//log.Infof("\tFixed version: %s\n", vulnerability.FixedBy)
			//fl.WriteString("\tFixed version: " + vulnerability.FixedBy + "\n")
			midResult = "\tFixed version: " + vulnerability.FixedBy + "\n"
			result += midResult
		}

		if vulnerability.Link != "" {
			//log.Infof("\tLink:          %s\n", vulnerability.Link)
			//fl.WriteString("\tLink:          " + vulnerability.Link + "\n")
			midResult = "\tLink:          " + vulnerability.Link + "\n"
			result += midResult
		}

		//log.Infof("\tLayer:         %s\n", feature.AddedBy)
		//		fl.WriteString("\tLayer:         " + feature.AddedBy + "\n")
		midResult = "\tLayer:         " + feature.AddedBy + "\n"
		result += midResult
		result += "\n"
		//log.Infof("")
		//fl.WriteString("\n")
	}

	if isSafe {
		//log.Infof("%s No vulnerabilities were detected in your image\n", ("Success!"))
		//fl.WriteString("Success!" + " No vulnerabilities were detected in your image\n")
		result = "Success!" + " No vulnerabilities were detected in your image\n"
	} else if !hasVisibleVulnerabilities {
		//log.Infof("%s No vulnerabilities matching the minimum severity level were detected in your image\n", ("NOTE:"))
		//fl.WriteString("NOTE:" + " No vulnerabilities matching the minimum severity level were detected in your image\n")
		result = "NOTE:" + " No vulnerabilities matching the minimum severity level were detected in your image\n"
	}
	//	defer fl.Close()
	return result, nil
}

func getImageSize(manifest distribution.Manifest) float64 {
	var totalSize int64
	totalSize = 0
	for i := 0; i < len(manifest.References()); i++ {
		log.Infof("~~~~~~~~~~~~~~~~~~layer:%d size:%d", i, manifest.References()[i].Size)
		totalSize += manifest.References()[i].Size
	}
	totalSizeMB := (float64)(totalSize) / (1024 * 1024)
	log.Infof("total size is %d , unit is %s", totalSizeMB, "MB")
	return totalSizeMB

}

/*
func createfile(repository string, tag string) (string, error) {
	str := strings.Split(repository, "/")

	if len(str) != 2 {
		log.Errorf("the repository's name is not standard")
		return "", errors.New("the repository's name is not standard")
	}
	project := str[0]
	image := str[1]
	dir := PATH + project
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		log.Errorf("can not create dir,error message%s : %s", err)
		return "", err
	}
	_, err = os.Create(dir + "/" + image + "_" + tag + ".log")
	if err != nil {
		log.Errorf("can not create txt,error message%s : %s", err)
		return "", err
	}
	return dir + "/" + image + "_" + tag + ".log", nil

}
*/
func ScanManifest(user string, manifest distribution.Manifest, repository string, tag string) (float64, string, error) {
	currentregistry = os.Getenv("HARBOR_URL")
	currentregistry = currentregistry + "/v2"
	clairURL = strings.Replace(os.Getenv("HARBOR_URL"),"s","",1)
	clairURL = clairURL + ":6060/v1"
	log.Info("%%%%%%%%%%%%%%%%%%%%%%%%%%%%% get image size start")
	image_size := getImageSize(manifest)

	log.Infof("**************************** size:%d", image_size)

	log.Info("%%%%%%%%%%%%%%%%%%%%%%%%%%%%% get image size end")
	log.Info("%%%%%%%%%%%%%%%%%%%%%%%%%%%%% new image start")
	image, err := newImage(currentregistry, repository, tag, user, manifest)
	if err != nil {
		log.Errorf("error occurred while getting manifest for %s:%s : %v", repository, tag, err)
		return image_size, "", err
	}
	log.Info("%%%%%%%%%%%%%%%%%%%%%%%%%%%%% new image end")
	/*
		log.Info("%%%%%%%%%%%%%%%%%%%%%%%%%%%%% create file start")
		file, err := createfile(repository, tag)
		if err != nil {
			log.Errorf("error occurred while getting manifest for %s:%s : %v", repository, tag, err)
			return image_size, "", err
		}
		log.Info("%%%%%%%%%%%%%%%%%%%%%%%%%%%%% create file end")
	*/
	c := &Clair{clairURL}
	result, err := c.analyse(image)
	if err != nil {
		log.Errorf("Error happens when anlysing : %v", err)
	}
	log.Info("%%%%%%%%%%%%%%%%%%%%%%%%%%%%% analyse end")
	return image_size, result, err
}

func filterEmptyLayers(fsLayers []FsLayer) (filteredLayers []FsLayer) {
	for _, layer := range fsLayers {
		if layer.BlobSum != EMPTY_LAYER_BLOB_SUM {
			filteredLayers = append(filteredLayers, layer)
		}
	}
	return
}
