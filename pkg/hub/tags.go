/*
   Copyright 2020 Docker Hub Tool authors

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

package hub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/distribution/reference"
	hubapi "github.com/docker/hub-tool/pkg/hub/api"
)

const (
	// TagsURL path to the Hub API listing the tags
	TagsURL = "/v2/namespaces/%s/repositories/%s/tags"
	// DeleteTagURL path to the Hub API to remove a tag
	DeleteTagURL = "/v2/namespaces/%s/repositories/%s/tags/%s"
)

// Tag can point to a manifest or manifest list
type Tag struct {
	Name                string
	FullSize            int
	LastUpdated         time.Time
	LastUpdaterUserName string
	Images              []Image
	LastPulled          time.Time
	LastPushed          time.Time
	Status              string
}

// Image represents the metadata of a manifest
type Image struct {
	Digest       string
	Architecture string
	Os           string
	Variant      string
	Size         int
	LastPulled   time.Time
	LastPushed   time.Time
	Status       string
}

// GetTags calls the hub repo API and returns all the information on all tags
func (c *Client) GetTags(repository string, reqOps ...RequestOp) ([]Tag, int, error) {
	namespace, name, err := getNamespaceRepository(repository)
	if err != nil {
		return nil, 0, err
	}
	rawURL, err := paginatedURL(c.domain+fmt.Sprintf(TagsURL, namespace, name), itemsPerPage)
	if err != nil {
		return nil, 0, err
	}

	tags, total, next, err := c.getTagsPage(rawURL, repository, reqOps...)
	if err != nil {
		return nil, 0, err
	}
	if c.fetchAllElements {
		for next != "" {
			pageTags, _, n, err := c.getTagsPage(next, repository, reqOps...)
			if err != nil {
				return nil, 0, err
			}
			next = n
			tags = append(tags, pageTags...)
		}
	}

	return tags, total, nil
}

// RemoveTag removes a tag in a repository on Hub
func (c *Client) RemoveTag(repository, tag string) error {
	namespace, name, err := getNamespaceRepository(repository)
	if err != nil {
		return err
	}
	return c.deleteHubResource(c.domain + fmt.Sprintf(DeleteTagURL, namespace, name, tag))
}

func (c *Client) getTagsPage(url, repository string, reqOps ...RequestOp) ([]Tag, int, string, error) {
	var hubResponse hubapi.PaginatedTags
	response, err := c.getTagPageResponse(url, reqOps...)
	if err != nil {
		return nil, 0, "", err
	}
	if err := json.Unmarshal(normalizeTagPageResponse(response), &hubResponse); err != nil {
		return nil, 0, "", err
	}
	var tags []Tag
	if hubResponse.Results == nil {
		return nil, ptrValue(hubResponse.Count), ptrValue(hubResponse.Next), nil
	}
	for _, result := range *hubResponse.Results {
		lastUpdated, err := parseAPITime(ptrValue(result.LastUpdated))
		if err != nil {
			return nil, 0, "", err
		}
		lastPulled, err := parseAPITime(ptrValue(result.TagLastPulled))
		if err != nil {
			return nil, 0, "", err
		}
		lastPushed, err := parseAPITime(ptrValue(result.TagLastPushed))
		if err != nil {
			return nil, 0, "", err
		}
		tag := Tag{
			Name:                fmt.Sprintf("%s:%s", repository, ptrValue(result.Name)),
			FullSize:            ptrValue(result.FullSize),
			LastUpdated:         lastUpdated,
			LastUpdaterUserName: ptrValue(result.LastUpdaterUsername),
			Images:              toImages(result.Images),
			Status:              string(ptrValue(result.Status)),
			LastPulled:          lastPulled,
			LastPushed:          lastPushed,
		}
		tags = append(tags, tag)
	}
	return tags, ptrValue(hubResponse.Count), ptrValue(hubResponse.Next), nil
}

func (c *Client) getTagPageResponse(url string, reqOps ...RequestOp) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.doRequest(req, append(reqOps, withHubToken(c.token))...)
}

func normalizeTagPageResponse(response []byte) []byte {
	var page map[string]interface{}
	if err := json.Unmarshal(response, &page); err != nil {
		return response
	}
	results, ok := page["results"].([]interface{})
	if !ok {
		return response
	}
	for _, result := range results {
		tag, ok := result.(map[string]interface{})
		if !ok {
			continue
		}
		normalizeTagResponse(tag)
	}
	normalized, err := json.Marshal(page)
	if err != nil {
		return response
	}
	return normalized
}

func normalizeTagResponse(tag map[string]interface{}) {
	if v2, ok := tag["v2"].(bool); ok {
		tag["v2"] = fmt.Sprintf("%v", v2)
	}
	if _, ok := tag["status"]; !ok {
		if status, ok := tag["tag_status"]; ok {
			tag["status"] = status
		}
	}
	if images, ok := tag["images"].([]interface{}); ok {
		if len(images) == 0 {
			delete(tag, "images")
			return
		}
		tag["images"] = images[0]
	}
}

func getRepoPath(s string) (string, error) {
	ref, err := reference.ParseNormalizedNamed(s)
	if err != nil {
		return "", err
	}
	ref = reference.TagNameOnly(ref)
	ref = reference.TrimNamed(ref)
	return reference.Path(ref), nil
}

func getNamespaceRepository(s string) (string, string, error) {
	repoPath, err := getRepoPath(s)
	if err != nil {
		return "", "", err
	}
	parts := strings.Split(repoPath, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid reference: expected namespace/repository")
	}
	return parts[0], parts[1], nil
}

func toImages(result *hubapi.Image) []Image {
	if result == nil {
		return nil
	}
	lastPulled, _ := parseAPITime(ptrValue(result.LastPulled))
	lastPushed, _ := parseAPITime(ptrValue(result.LastPushed))
	return []Image{{
		Digest:       ptrValue(result.Digest),
		Architecture: ptrValue(result.Architecture),
		Os:           ptrValue(result.Os),
		Variant:      ptrValue(result.Variant),
		Size:         ptrValue(result.Size),
		Status:       string(ptrValue(result.Status)),
		LastPulled:   lastPulled,
		LastPushed:   lastPushed,
	}}
}
