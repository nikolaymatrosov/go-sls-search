package main

import (
	"cdk.tf/go/stack/generated/yandex-cloud/yandex/functionresource"
	"cdk.tf/go/stack/generated/yandex-cloud/yandex/iamserviceaccount"
	sk "cdk.tf/go/stack/generated/yandex-cloud/yandex/iamserviceaccountstaticaccesskey"
	"cdk.tf/go/stack/generated/yandex-cloud/yandex/provider"
	rfm "cdk.tf/go/stack/generated/yandex-cloud/yandex/resourcemanagerfolderiammember"
	"cdk.tf/go/stack/generated/yandex-cloud/yandex/storageobject"
	"fmt"
	"github.com/aws/constructs-go/constructs/v10"
	j "github.com/aws/jsii-runtime-go"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
	"os"
	"path"
)

func NewMyStack(scope constructs.Construct, id string) cdktf.TerraformStack {
	stack := cdktf.NewTerraformStack(scope, &id)

	folderId := os.Getenv("FOLDER_ID")

	saFileContent, _ := os.ReadFile("./key.json")
	saFileContentStr := string(saFileContent)

	provider.NewYandexProvider(stack, j.String("provider"), &provider.YandexProviderConfig{
		ServiceAccountKeyFile: &saFileContentStr,
		FolderId:              &folderId,
	})

	sa := iamserviceaccount.NewIamServiceAccount(stack, j.String("go-sls-search-sa"),
		&iamserviceaccount.IamServiceAccountConfig{
			Name: j.String("go-sls-search-sa"),
		})

	rfm.NewResourcemanagerFolderIamMember(stack, j.String("s3-viewer"),
		&rfm.ResourcemanagerFolderIamMemberConfig{
			FolderId: &folderId,
			Role:     j.String("storage.viewer"),
			Member:   j.String(fmt.Sprintf("serviceAccount:%s", *sa.Id())),
		})

	uploader := iamserviceaccount.NewIamServiceAccount(stack, j.String("go-sls-search-uploader"),
		&iamserviceaccount.IamServiceAccountConfig{
			Name: j.String("go-sls-uploader-sa"),
		})

	rfm.NewResourcemanagerFolderIamMember(stack, j.String("s3-uploader"),
		&rfm.ResourcemanagerFolderIamMemberConfig{
			FolderId: &folderId,
			Role:     j.String("storage.uploader"),
			Member:   j.String(fmt.Sprintf("serviceAccount:%s", *uploader.Id())),
		})

	cwd, _ := os.Getwd()

	asset := cdktf.NewTerraformAsset(stack, j.String("go-sls-search-asset"), &cdktf.TerraformAssetConfig{
		Path: j.String(path.Join(cwd, "../src")),
		Type: cdktf.AssetType_ARCHIVE,
	})

	staticAccessKey := sk.NewIamServiceAccountStaticAccessKey(
		stack,
		j.String("go-sls-search-key"),
		&sk.IamServiceAccountStaticAccessKeyConfig{
			ServiceAccountId: sa.Id(),
			Description:      j.String("Key for fetching index to go-sls-search func"),
		})
	uploaderKey := sk.NewIamServiceAccountStaticAccessKey(
		stack,
		j.String("go-sls-uploader-key"),
		&sk.IamServiceAccountStaticAccessKeyConfig{
			ServiceAccountId: uploader.Id(),
			Description:      j.String("Key for uploading index to go-sls-search func"),
		})

	storageobject.NewStorageObject(
		stack,
		j.String("serialized-index-archive"),
		&storageobject.StorageObjectConfig{
			Bucket:    j.String("sls-search"),
			Key:       j.String("bleve.zip"),
			AccessKey: uploaderKey.AccessKey(),
			SecretKey: uploaderKey.SecretKey(),
			Source:    j.String(path.Join(cwd, "../data/anek/bleve.zip")),
		},
	)

	functionresource.NewFunctionResource(stack, j.String("go-sls-search"), &functionresource.FunctionResourceConfig{
		Content: &functionresource.FunctionResourceContent{
			ZipFilename: asset.Path(),
		},
		Entrypoint:       j.String("handler.SearchHandler"),
		ExecutionTimeout: j.String("5"),
		Memory:           j.Number(128),
		Name:             j.String("go-sls-search"),
		Runtime:          j.String("golang119"),
		UserHash:         asset.AssetHash(),
		Environment: &map[string]*string{
			"AWS_ACCESS_KEY_ID":     staticAccessKey.AccessKey(),
			"AWS_SECRET_ACCESS_KEY": staticAccessKey.SecretKey(),
		},
		Tags: j.Strings("zst"),
		//DependsOn: &[]cdktf.ITerraformDependable{index},
	})

	//functionresource.NewFunctionResource(stack, j.String("downloader"), &functionresource.FunctionResourceConfig{
	//	Content: &functionresource.FunctionResourceContent{
	//		ZipFilename: asset.Path(),
	//	},
	//	Entrypoint:       j.String("speed.SpeedHandler"),
	//	ExecutionTimeout: j.String("5"),
	//	Memory:           j.Number(128),
	//	Name:             j.String("downloader"),
	//	Runtime:          j.String("golang119"),
	//	UserHash:         asset.AssetHash(),
	//	Environment: &map[string]*string{
	//		"AWS_ACCESS_KEY_ID":     staticAccessKey.AccessKey(),
	//		"AWS_SECRET_ACCESS_KEY": staticAccessKey.SecretKey(),
	//	},
	//	//DependsOn: &[]cdktf.ITerraformDependable{index},
	//})

	return stack
}

func main() {
	app := cdktf.NewApp(nil)

	NewMyStack(app, "go-sls-search")

	app.Synth()
}
