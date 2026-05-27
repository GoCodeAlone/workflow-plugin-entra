package internal

import (
	"fmt"

	"github.com/GoCodeAlone/workflow-plugin-entra/internal/contracts"
	pb "github.com/GoCodeAlone/workflow/plugin/external/proto"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

var Version = "0.0.0"

type entraPlugin struct{}

func NewEntraPlugin() sdk.PluginProvider {
	return &entraPlugin{}
}

func (p *entraPlugin) Manifest() sdk.PluginManifest {
	return sdk.PluginManifest{
		Name:        "workflow-plugin-entra",
		Version:     Version,
		Author:      "GoCodeAlone",
		Description: "Microsoft Entra ID provider plugin backed by Microsoft Graph and Azure Identity SDKs",
	}
}

func (p *entraPlugin) ModuleTypes() []string {
	return []string{"entra.provider"}
}

func (p *entraPlugin) TypedModuleTypes() []string {
	return p.ModuleTypes()
}

func (p *entraPlugin) CreateModule(typeName, name string, config map[string]any) (sdk.ModuleInstance, error) {
	switch typeName {
	case "entra.provider":
		return newEntraModule(name, config)
	default:
		return nil, fmt.Errorf("entra plugin: unknown module type %q", typeName)
	}
}

func (p *entraPlugin) CreateTypedModule(typeName, name string, config *anypb.Any) (sdk.ModuleInstance, error) {
	if typeName != "entra.provider" {
		return nil, fmt.Errorf("entra plugin: unknown typed module type %q", typeName)
	}
	factory := sdk.NewTypedModuleFactory(typeName, &contracts.ProviderConfig{}, func(name string, cfg *contracts.ProviderConfig) (sdk.ModuleInstance, error) {
		return newEntraModule(name, typedModuleConfig(cfg))
	})
	return factory.CreateTypedModule(typeName, name, config)
}

func (p *entraPlugin) StepTypes() []string {
	return allStepTypes()
}

func (p *entraPlugin) TypedStepTypes() []string {
	return p.StepTypes()
}

func (p *entraPlugin) CreateStep(typeName, name string, config map[string]any) (sdk.StepInstance, error) {
	return createStep(typeName, name, config)
}

func (p *entraPlugin) CreateTypedStep(typeName, name string, config *anypb.Any) (sdk.StepInstance, error) {
	if _, ok := stepRegistry[typeName]; !ok {
		return nil, fmt.Errorf("%w: step type %q", sdk.ErrTypedContractNotHandled, typeName)
	}
	if typeName == "step.entra_auth_provider_describe" {
		return sdk.NewTypedStepFactory(typeName, &contracts.AuthProviderDescribeConfig{}, &contracts.AuthProviderDescribeInput{}, typedAuthProviderDescribe).CreateTypedStep(typeName, name, config)
	}
	return sdk.NewTypedStepFactory(typeName, &contracts.EntraStepConfig{}, &contracts.EntraStepInput{}, typedStepHandler(typeName)).CreateTypedStep(typeName, name, config)
}

func (p *entraPlugin) ContractRegistry() *pb.ContractRegistry {
	return contractRegistry
}

var contractRegistry = &pb.ContractRegistry{
	FileDescriptorSet: &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			protodesc.ToFileDescriptorProto(structpb.File_google_protobuf_struct_proto),
			protodesc.ToFileDescriptorProto(contracts.File_internal_contracts_entra_proto),
		},
	},
	Contracts: contractDescriptors(),
}

func contractDescriptors() []*pb.ContractDescriptor {
	descriptors := []*pb.ContractDescriptor{
		moduleContract("entra.provider", "ProviderConfig"),
	}
	for _, stepType := range allStepTypes() {
		if stepType == "step.entra_auth_provider_describe" {
			descriptors = append(descriptors, stepContract(stepType, "AuthProviderDescribeConfig", "AuthProviderDescribeInput", "AuthProviderDescribeOutput"))
			continue
		}
		descriptors = append(descriptors, stepContract(stepType, "EntraStepConfig", "EntraStepInput", "EntraStepOutput"))
	}
	return descriptors
}

func moduleContract(moduleType, configMessage string) *pb.ContractDescriptor {
	const pkg = "workflow.plugins.entra.v1."
	return &pb.ContractDescriptor{
		Kind:          pb.ContractKind_CONTRACT_KIND_MODULE,
		ModuleType:    moduleType,
		ConfigMessage: pkg + configMessage,
		Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
	}
}

func stepContract(stepType, configMessage, inputMessage, outputMessage string) *pb.ContractDescriptor {
	const pkg = "workflow.plugins.entra.v1."
	return &pb.ContractDescriptor{
		Kind:          pb.ContractKind_CONTRACT_KIND_STEP,
		StepType:      stepType,
		ConfigMessage: pkg + configMessage,
		InputMessage:  pkg + inputMessage,
		OutputMessage: pkg + outputMessage,
		Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
	}
}
