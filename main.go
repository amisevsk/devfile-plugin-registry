package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	meta "github.com/amisevsk/devworkspace-conversion/meta_yaml"
	devfile "github.com/devfile/api/pkg/apis/workspaces/v1alpha2"
	brokerModel "github.com/eclipse/che-plugin-broker/model"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/yaml"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(devfile.AddToScheme(scheme))
}

func main() {
	var inputPath, outputPath string
	flag.StringVar(&inputPath, "from", "undefined", "Path to input meta.yaml")
	flag.StringVar(&outputPath, "to", "undefined", "Path to output devworkspacetemplate.yaml")
	flag.Parse()
	if inputPath == "undefined" || outputPath == "undefined" {
		fmt.Println("Arguments --from and --to are required")
		os.Exit(1)
	}
	pluginMeta, err := readPluginMetaFromFile(inputPath)
	if err != nil {
		fmt.Printf("Error reading input file: %s\n", err)
		os.Exit(1)
	}
	devfile, err := meta.ConvertMetaYamlToDevfile(pluginMeta)
	if err != nil {
		fmt.Printf("Error converting plugin meta.yaml to DevWorkspaceTemplate: %s\n", err)
		os.Exit(1)
	}
	err = writeDevfileToFile(devfile, outputPath)
	if err != nil {
		fmt.Printf("Error writing output file: %s\n", err)
		os.Exit(1)
	}
}

func readPluginMetaFromFile(path string) (*brokerModel.PluginMeta, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read DevWorkspaceTemplate at path %s: %w", path, err)
	}
	pluginMeta := &brokerModel.PluginMeta{}
	err = yaml.Unmarshal(bytes, pluginMeta)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal DevWorkspaceTemplate: %w", err)
	}
	return pluginMeta, nil
}

func writeDevfileToFile(devfile *meta.Devfile, path string) error {
	bytes, err := yaml.Marshal(devfile)
	if err != nil {
		return fmt.Errorf("failed to serialize devfile: %w", err)
	}
	err = ioutil.WriteFile(path, bytes, 0777)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}
	return nil
}
