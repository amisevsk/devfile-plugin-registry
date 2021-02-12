package meta

import (
	"fmt"
	"strconv"

	devfilev2 "github.com/devfile/api/pkg/devfile"

	dw "github.com/devfile/api/pkg/apis/workspaces/v1alpha2"
	devfileAttributes "github.com/devfile/api/pkg/attributes"
	brokerModel "github.com/eclipse/che-plugin-broker/model"
)

const (
	metaSecureAttribute    = "secure"
	metaProtocolAttribute  = "protocol"
	extensionsAttributeKey = "che-theia.eclipse.org/vscode-extensions"
)

type Devfile struct {
	devfilev2.DevfileHeader
	dw.DevWorkspaceTemplateSpec
}

func ConvertMetaYamlToDevfile(meta *brokerModel.PluginMeta) (*Devfile, error) {
	if len(meta.Spec.InitContainers) > 0 {
		return nil, fmt.Errorf("initContainers not supported for automatic conversion")
	}
	if len(meta.Spec.WorkspaceEnv) > 0 {
		// No plugins currently use workspaceEnv
		return nil, fmt.Errorf("workspaceEnv is not supported for automatic conversion")
	}
	if len(meta.Spec.Containers) > 1 {
		return nil, fmt.Errorf("only zero or one containers are supported")
	}
	devfile := &Devfile{
		DevfileHeader: devfilev2.DevfileHeader{
			SchemaVersion: "2.0.0",
			Metadata: devfilev2.DevfileMetadata{
				Name:       meta.Name,
				Version:    meta.Version,
				Attributes: getAttributesFromMeta(meta),
			},
		},
	}
	component := dw.Component{}
	component.Name = meta.Name
	if len(meta.Spec.Containers) > 0 {
		component.Container = convertMetaToContainer(meta.Spec.Containers[0])
		endpoints, err := convertMetaEndpoints(meta.Spec.Endpoints)
		if err != nil {
			return nil, fmt.Errorf("failed to convert meta endpoints: %w", err)
		}
		component.Container.Endpoints = endpoints
		devfile.Components = append(devfile.Components, getVolumeComponentsForVolumeMounts(component.Container.VolumeMounts)...)
		appendRequiredEnvWorkaround(meta.Name, component.Container)
		appendRequiredVolumeMounts(component.Container)
	}
	if len(meta.Spec.Extensions) > 0 {
		var err error
		if component.Attributes == nil {
			component.Attributes = devfileAttributes.Attributes{}
		}
		component.Attributes.Put(extensionsAttributeKey, meta.Spec.Extensions, &err)
		if err != nil {
			return nil, fmt.Errorf("failed parsing extensions for plugin %s: %w", meta.Name, err)
		}
	}
	devfile.Components = append(devfile.Components, component)

	return devfile, nil
}

func getAttributesFromMeta(meta *brokerModel.PluginMeta) devfileAttributes.Attributes {
	attrs := devfileAttributes.Attributes{}
	attrs.Put("meta.yaml", map[string]string{
		"name":        meta.Name,
		"publisher":   meta.Publisher,
		"version":     meta.Version,
		"id":          fmt.Sprintf("%s/%s/%s", meta.Publisher, meta.Name, meta.Version),
		"type":        meta.Type,
		"displayName": meta.DisplayName,
		"title":       meta.Title,
		"description": meta.Description,
	}, nil)
	return attrs
}

func convertMetaToContainer(metaContainer brokerModel.Container) *dw.ContainerComponent {
	boolTrue := true
	container := &dw.ContainerComponent{}
	container.Image = metaContainer.Image
	container.Command = metaContainer.Command
	container.Args = metaContainer.Args
	container.MountSources = &boolTrue
	// TODO: Only limit supported for now; need to update dependency to include https://github.com/devfile/api/pull/318
	container.MemoryLimit = metaContainer.MemoryLimit
	for _, env := range metaContainer.Env {
		container.Env = append(container.Env, dw.EnvVar{
			Name:  env.Name,
			Value: env.Value,
		})
	}
	for _, vm := range metaContainer.Volumes {
		container.VolumeMounts = append(container.VolumeMounts, dw.VolumeMount{
			Name: vm.Name,
			Path: vm.MountPath,
			// TODO: Update devfile/api dep to pull in ephemeral support
		})
	}
	return container
}

func convertMetaEndpoints(endpoints []brokerModel.Endpoint) ([]dw.Endpoint, error) {
	var devfileEndpoints []dw.Endpoint
	for _, endpoint := range endpoints {
		exposure := dw.PublicEndpointExposure
		if endpoint.Public == false {
			exposure = dw.InternalEndpointExposure
		}
		secure := false
		if secureVal, ok := endpoint.Attributes[metaSecureAttribute]; ok {
			var err error
			secure, err = strconv.ParseBool(secureVal)
			if err != nil {
				return nil, fmt.Errorf("failed to parse value for 'secure' attribute: %w", err)
			}
		}
		protocol := dw.HTTPEndpointProtocol
		if protocolVal, ok := endpoint.Attributes[metaProtocolAttribute]; ok {
			protocol = dw.EndpointProtocol(protocolVal)
		}
		attributes := devfileAttributes.Attributes{}
		attributes = attributes.FromStringMap(endpoint.Attributes)
		devfileEndpoints = append(devfileEndpoints, dw.Endpoint{
			Name:       endpoint.Name,
			TargetPort: endpoint.TargetPort,
			Exposure:   exposure,
			Protocol:   protocol,
			Secure:     secure,
			Attributes: attributes,
		})
	}
	return devfileEndpoints, nil
}

func getVolumeComponentsForVolumeMounts(vms []dw.VolumeMount) []dw.Component {
	var components []dw.Component
	for _, vm := range vms {
		components = append(components, dw.Component{
			Name: vm.Name,
			ComponentUnion: dw.ComponentUnion{
				Volume: &dw.VolumeComponent{},
			},
		})
	}
	return components
}

func appendRequiredEnvWorkaround(metaName string, container *dw.ContainerComponent) {
	container.Env = append(container.Env, dw.EnvVar{
		Name:  "PLUGIN_REMOTE_ENDPOINT_EXECUTABLE",
		Value: "/remote-endpoint/plugin-remote-endpoint",
	}, dw.EnvVar{
		Name:  "THEIA_PLUGINS",
		Value: fmt.Sprintf("local-dir:///plugins/sidecars/%s", metaName),
	})
}

func appendRequiredVolumeMounts(container *dw.ContainerComponent) {
	// These volumes are provided by Theia
	container.VolumeMounts = append(container.VolumeMounts, dw.VolumeMount{
		Name: "remote-endpoint",
		Path: "/remote-endpoint",
	}, dw.VolumeMount{
		Name: "plugins",
		Path: "/plugins",
	})
}
