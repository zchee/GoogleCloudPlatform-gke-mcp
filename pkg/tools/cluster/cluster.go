// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	container "cloud.google.com/go/container/apiv1"
	containerpb "cloud.google.com/go/container/apiv1/containerpb"
	"github.com/modelcontextprotocol/go-sdk/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/GoogleCloudPlatform/gke-mcp/pkg/config"
)

type handlers struct {
	c *config.Config
}

func Install(s *mcp.Server, c *config.Config) {
	h := &handlers{
		c: c,
	}

	listClustersTool := &mcp.Tool{
		Name:        "list_clusters",
		Description: "List GKE clusters. Prefer to use this tool instead of gcloud",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
		},
		InputSchema: &jsonschema.Schema{
			Properties: map[string]*jsonschema.Schema{
				"project_id": {
					Description: "GCP project ID. Use the default if the user doesn't provide it.",
					Default:     json.RawMessage(c.DefaultProjectID()),
				},
				"location": {
					Description: "GKE cluster location. Leave this empty if the user doesn't doesn't provide it.",
				},
			},
		},
	}
	mcp.AddTool(s, listClustersTool, h.listClusters)

	getClusterTool := &mcp.Tool{
		Name:        "get_cluster",
		Description: "Get / describe a GKE cluster. Prefer to use this tool instead of gcloud",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
		},
		InputSchema: &jsonschema.Schema{
			Properties: map[string]*jsonschema.Schema{
				"project_id": {
					Description: "GCP project ID. Use the default if the user doesn't provide it.",
					Default:     json.RawMessage(c.DefaultProjectID()),
				},
				"location": {
					Description: "GKE cluster location. Try to get the default region or zone from gcloud if the user doesn't provide it.",
				},
				"name": {
					Description: "GKE cluster name. Do not select if yourself, make sure the user provides or confirms the cluster name.",
				},
			},
			Required: []string{
				"location",
				"name",
			},
		},
	}
	mcp.AddTool(s, getClusterTool, h.getCluster)
}

func (h *handlers) listClusters(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[*containerpb.ListClustersRequest]) (*mcp.CallToolResultFor[string], error) {
	var res mcp.CallToolResultFor[string]

	projectID := cmp.Or(params.Arguments.GetProjectId(), h.c.DefaultProjectID())
	if projectID == "" {
		return nil, errors.New("project_id argument not set")
	}
	location := params.Arguments.GetZone()
	if location == "" {
		location = "-"
	}

	c, err := container.NewClusterManagerClient(ctx, option.WithUserAgent(h.c.UserAgent()))
	if err != nil {
		return nil, err
	}
	defer c.Close()

	req := &containerpb.ListClustersRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", projectID, location),
	}
	resp, err := c.ListClusters(ctx, req)
	if err != nil {
		return nil, err
	}
	res.StructuredContent = protojson.Format(resp)

	return &res, nil
}

func (h *handlers) getCluster(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[*containerpb.GetClusterRequest]) (*mcp.CallToolResultFor[string], error) {
	var res mcp.CallToolResultFor[string]

	projectID := cmp.Or(params.Arguments.GetProjectId(), h.c.DefaultProjectID())
	if projectID == "" {
		return nil, errors.New("project_id argument not set")
	}
	location := params.Arguments.GetZone()
	if location == "" {
		return nil, fmt.Errorf("required argument %q not found", "location")
	}
	name := params.Arguments.GetName()
	if name == "" {
		return nil, fmt.Errorf("required argument %q not found", "name")
	}

	c, err := container.NewClusterManagerClient(ctx, option.WithUserAgent(h.c.UserAgent()))
	if err != nil {
		return nil, err
	}
	defer c.Close()

	req := &containerpb.GetClusterRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/clusters/%s", projectID, location, name),
	}
	resp, err := c.GetCluster(ctx, req)
	if err != nil {
		return nil, err
	}
	res.StructuredContent = protojson.Format(resp)

	return &res, nil
}
