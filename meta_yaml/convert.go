package meta

import (
	"fmt"
	"strconv"

	devfile "github.com/devfile/api/pkg/apis/workspaces/v1alpha2"
	devfileAttributes "github.com/devfile/api/pkg/attributes"
	brokerModel "github.com/eclipse/che-plugin-broker/model"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	metaSecureAttribute    = "secure"
	metaProtocolAttribute  = "protocol"
	extensionsAttributeKey = "che-theia.eclipse.org/vscode-extensions"
)

func ConvertMetaYamlToDevWorkspaceTemplate(meta *brokerModel.PluginMeta) (*devfile.DevWorkspaceTemplate, error) {
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
	dwt := &devfile.DevWorkspaceTemplate{}
	component := devfile.Component{}
	component.Name = meta.Name
	if len(meta.Spec.Containers) > 0 {
		component.Container = convertMetaToContainer(meta.Spec.Containers[0])
		endpoints, err := convertMetaEndpoints(meta.Spec.Endpoints)
		if err != nil {
			return nil, fmt.Errorf("failed to convert meta endpoints: %w", err)
		}
		component.Container.Endpoints = endpoints
		dwt.Spec.Components = append(dwt.Spec.Components, getVolumeComponentsForVolumeMounts(component.Container.VolumeMounts)...)
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
	dwt.Spec.Components = append(dwt.Spec.Components, component)
	dwt.Name = meta.Name
	dwt.Annotations = getAnnotationsFromMeta(meta)
	dwt.TypeMeta = v1.TypeMeta{
		Kind:       "DevWorkspaceTemplate",
		APIVersion: "workspace.devfile.io/v1alpha2",
	}
	return dwt, nil
}

func convertMetaToContainer(metaContainer brokerModel.Container) *devfile.ContainerComponent {
	boolTrue := true
	container := &devfile.ContainerComponent{}
	container.Image = metaContainer.Image
	container.Command = metaContainer.Command
	container.Args = metaContainer.Args
	container.MountSources = &boolTrue
	// TODO: Only limit supported for now; need to update dependency to include https://github.com/devfile/api/pull/318
	container.MemoryLimit = metaContainer.MemoryLimit
	for _, env := range metaContainer.Env {
		container.Env = append(container.Env, devfile.EnvVar{
			Name:  env.Name,
			Value: env.Value,
		})
	}
	for _, vm := range metaContainer.Volumes {
		container.VolumeMounts = append(container.VolumeMounts, devfile.VolumeMount{
			Name: vm.Name,
			Path: vm.MountPath,
			// TODO: Update devfile/api dep to pull in ephemeral support
		})
	}

	return container
}

func convertMetaEndpoints(endpoints []brokerModel.Endpoint) ([]devfile.Endpoint, error) {
	var devfileEndpoints []devfile.Endpoint
	for _, endpoint := range endpoints {
		exposure := devfile.PublicEndpointExposure
		if endpoint.Public == false {
			exposure = devfile.InternalEndpointExposure
		}
		secure := false
		if secureVal, ok := endpoint.Attributes[metaSecureAttribute]; ok {
			var err error
			secure, err = strconv.ParseBool(secureVal)
			if err != nil {
				return nil, fmt.Errorf("failed to parse value for 'secure' attribute: %w", err)
			}
		}
		protocol := devfile.HTTPEndpointProtocol
		if protocolVal, ok := endpoint.Attributes[metaProtocolAttribute]; ok {
			protocol = devfile.EndpointProtocol(protocolVal)
		}
		attributes := devfileAttributes.Attributes{}
		attributes = attributes.FromStringMap(endpoint.Attributes)
		devfileEndpoints = append(devfileEndpoints, devfile.Endpoint{
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

func getVolumeComponentsForVolumeMounts(vms []devfile.VolumeMount) []devfile.Component {
	var components []devfile.Component
	for _, vm := range vms {
		components = append(components, devfile.Component{
			Name: vm.Name,
			ComponentUnion: devfile.ComponentUnion{
				Volume: &devfile.VolumeComponent{},
			},
		})
	}
	return components
}

func getAnnotationsFromMeta(meta *brokerModel.PluginMeta) map[string]string {
	return map[string]string{
		"che.eclipse.org/plugin/name":         meta.Name,
		"che.eclipse.org/plugin/publisher":    meta.Publisher,
		"che.eclipse.org/plugin/version":      meta.Version,
		"che.eclipse.org/plugin/display-name": meta.DisplayName,
		"che.eclipse.org/plugin/type":         meta.Type,
		"che.eclipse.org/plugin/description":  meta.Description,
	}
}
