package kit

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Shopify/themekit/kittest"
)

func TestIsValid(t *testing.T) {
	asset := Asset{Key: "test.txt", Value: "one"}
	assert.Equal(t, true, asset.IsValid())
	asset = Asset{Key: "test.txt", Attachment: "one"}
	assert.Equal(t, true, asset.IsValid())
	asset = Asset{Value: "one"}
	assert.Equal(t, false, asset.IsValid())
	asset = Asset{Key: "test.txt"}
	assert.Equal(t, false, asset.IsValid())
}

func TestSize(t *testing.T) {
	asset := Asset{Value: "one"}
	assert.Equal(t, 3, asset.Size())
	asset = Asset{Attachment: "other"}
	assert.Equal(t, 5, asset.Size())
}

func TestWrite(t *testing.T) {
	kittest.Setup()
	defer kittest.Cleanup()
	asset := Asset{Key: "output/blah.txt", Value: "this is content"}
	assert.NotNil(t, asset.Write("./does/not/exist"))
	assert.Nil(t, asset.Write(kittest.FixtureProjectPath))
}

func TestContents(t *testing.T) {
	asset := Asset{Value: "this is content"}
	data, err := asset.Contents()
	assert.Nil(t, err)
	assert.Equal(t, 15, len(data))

	asset = Asset{Attachment: "this is bad content"}
	data, err = asset.Contents()
	assert.NotNil(t, err)

	asset = Asset{Attachment: base64.StdEncoding.EncodeToString([]byte("this is bad content"))}
	data, err = asset.Contents()
	assert.Nil(t, err)
	assert.Equal(t, 19, len(data))
	assert.Equal(t, []byte("this is bad content"), data)

	asset = Asset{Key: "test.json", Value: "{\"test\":\"one\"}"}
	data, err = asset.Contents()
	assert.Nil(t, err)
	assert.Equal(t, 19, len(data))
	assert.Equal(t, `{
  "test": "one"
}`, string(data))
}

func TestFindAllFiles(t *testing.T) {
	kittest.Setup()
	kittest.GenerateProject()
	defer kittest.Cleanup()
	files, err := findAllFiles(kittest.ProjectFiles[0])
	assert.Equal(t, "Path is not a directory", err.Error())
	files, err = findAllFiles(kittest.FixtureProjectPath)
	assert.Nil(t, err)
	assert.Equal(t, len(kittest.ProjectFiles), len(files))
}

func TestLoadAssetsFromDirectory(t *testing.T) {
	kittest.Setup()
	kittest.GenerateProject()
	defer kittest.Cleanup()

	assets, err := loadAssetsFromDirectory(kittest.ProjectFiles[0], "", func(path string) bool { return false })
	assert.Equal(t, "Path is not a directory", err.Error())
	assets, err = loadAssetsFromDirectory(kittest.FixtureProjectPath, "", func(path string) bool {
		return path != "assets/application.js"
	})
	assert.Nil(t, err)
	assert.Equal(t, []Asset{{
		Key:   "assets/application.js",
		Value: "this is js content",
	}}, assets)

	kittest.Setup()
	kittest.GenerateProject()
	defer kittest.Cleanup()
	assets, err = loadAssetsFromDirectory(kittest.FixtureProjectPath, "assets", func(path string) bool { return false })
	assert.Nil(t, err)
	assert.Equal(t, 2, len(assets))
}

func TestLoadAsset(t *testing.T) {
	kittest.Setup()
	kittest.GenerateProject()
	defer kittest.Cleanup()

	asset, err := loadAsset(kittest.FixtureProjectPath, kittest.ProjectFiles[0])
	assert.Equal(t, kittest.ProjectFiles[0], asset.Key)
	assert.Equal(t, true, asset.IsValid())
	assert.Equal(t, "this is js content", asset.Value)
	assert.Nil(t, err)

	asset, err = loadAsset(kittest.FixtureProjectPath, "nope.txt")
	assert.NotNil(t, err)

	asset, err = loadAsset(kittest.FixtureProjectPath, "templates")
	assert.NotNil(t, err)
	assert.Equal(t, ErrAssetIsDir, err)

	asset, err = loadAsset(kittest.FixtureProjectPath, "assets/pixel.png")
	assert.Nil(t, err)
	assert.True(t, len(asset.Attachment) > 0)
	assert.True(t, asset.IsValid())
}
