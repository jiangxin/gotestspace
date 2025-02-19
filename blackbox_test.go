package testspace_test

import (
	"context"
	"os"
	"path"
	"strings"
	"testing"

	testspace "github.com/Jiu2015/gotestspace"
	"github.com/stretchr/testify/assert"
)

// The sample test for running "printf hello"
func TestSampleShell(t *testing.T) {
	assert := assert.New(t)
	workspace, err := testspace.Create(testspace.WithShellOption("printf \"hello\""))
	if !assert.NoError(err) {
		assert.FailNowf("create testspace got error", "%v", err)
	}
	defer workspace.Cleanup()

	assert.Equal("hello", workspace.GetOutputStr())
}

// Add environment example
func TestSampleShellWithEnvironments(t *testing.T) {
	assert := assert.New(t)
	workspace, err := testspace.Create(
		// Add two environments Testing1 and Testing2
		testspace.WithEnvironmentsOption("Testing1=aa", "Testing2=bb"),
		testspace.WithShellOption("printf \"%s, %s\" $Testing1 $Testing2"),
	)
	if !assert.NoError(err) {
		assert.FailNowf("create testspace got error", "%v", err)
	}
	defer workspace.Cleanup()

	assert.Equal("aa, bb", workspace.GetOutputStr())
}

// Add template example
func TestAddTemplateAndCall(t *testing.T) {
	assert := assert.New(t)
	workspace, err := testspace.Create(
		testspace.WithTemplateOption(`
test(){
	printf "this is a test from test method"
}
`),
		testspace.WithShellOption("test"))
	if !assert.NoError(err) {
		assert.FailNowf("create testspace got error", "%v", err)
	}
	defer workspace.Cleanup()

	assert.Equal("this is a test from test method", workspace.GetOutputStr())
}

// Add custom path example
func TestSetCustomPathForTesting(t *testing.T) {
	assert := assert.New(t)
	currentPath, _ := os.Getwd()
	testFolderName := "testing_folder"
	testPath := path.Join(currentPath, testFolderName)
	workspace, err := testspace.Create(
		testspace.WithPathOption(testPath),
	)
	if !assert.NoError(err) {
		assert.FailNowf("create testspace got error", "%v", err)
	}
	defer workspace.Cleanup()

	workPath := workspace.GetPath(testFolderName)

	_, err = os.Stat(workPath)
	if err != nil {
		assert.Error(err, "the custom path not exist")
	}
}

// Create a bare repository example
func TestCreateBareRepository(t *testing.T) {
	assert := assert.New(t)

	// The "test_tick" is the default method in template
	workspace, err := testspace.Create(
		testspace.WithShellOption(`
git init --bare test.git &&
git clone test.git test && 
(
	cd test && 
	printf "this is a test\n">init.js &&
	git add init.js &&
	test_tick &&
	git commit -m "this is the first commit" &&
	git push 
) &&
rm -rf test
`))
	if !assert.NoError(err) {
		assert.FailNowf("create testspace got error", "%s", workspace.GetOutputStr())
	}
	defer workspace.Cleanup()

	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Let's add the second commit, running custom shell again
	_, _, err = workspace.Execute(cancelCtx, `
git clone test.git test && 
(
	cd test && 
	printf "add a new file\n"> main.go &&
	git add main.go && 
	test_tick &&
	git commit -m "this is the second commit" && 
	git push 
)&&
rm -rf test
`)
	if !assert.NoError(err, "create testspace got error") {
		assert.FailNowf("create testspace got error", "%s", workspace.GetOutputStr())
	}

	// Now, let's check the bare repository
	_, _, err = workspace.Execute(cancelCtx, "cd test.git && git log --oneline")
	if !assert.NoError(err) {
		assert.FailNowf("create testspace got error", "%v", err)
	}

	assert.Equal("95dbed8 this is the second commit\n5a1f64b this is the first commit",
		strings.TrimSpace(workspace.GetOutputStr()))
}

// Test use stdin to delete bare repository branch
func TestDeleteBareRepositoryBranch(t *testing.T) {
	assert := assert.New(t)
	// The "test_tick" is the default method in template
	workspace, err := testspace.Create(
		testspace.WithShellOption(`
git init --bare test.git &&
git clone test.git test && 
(
	cd test && 
	printf "this is a test">init.js &&
	git add init.js &&
	test_tick &&
	git commit -m "this is the first commit" &&
	git push &&
	printf "the js file content">resource.js &&
	git add resource.js &&
	test_tick &&
	git commit -m "the second commit" && 
	git branch branch1 &&
	printf "the go file">main.go &&
	git add main.go &&
	test_tick &&
	git commit -m "the third commit" &&
	git branch branch2 &&
	git push --all
) &&
rm -rf test
`))
	if !assert.NoError(err) {
		assert.FailNowf("create testspace got error", "%v", err)
	}
	defer workspace.Cleanup()

	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	command, err := workspace.ExecuteWithStdin(cancelCtx, "cd test.git && git update-ref --stdin")
	if !assert.NoError(err) {
		assert.FailNowf("execute with stdin got error", "%v", err)
	}

	// Start delete branch1
	command.Write([]byte("delete refs/heads/branch1\n"))

	// Start delete branch2
	command.Write([]byte("delete refs/heads/branch2\n"))

	if err = command.Wait(); !assert.NoError(err) {
		assert.FailNowf("execute with stdin got error", "%v", err)
	}

	// Let's check all the branches
	stdout, _, err := workspace.Execute(cancelCtx, "cd test.git && git branch --format=\"%(refname)\"")
	if !assert.NoError(err) {
		assert.FailNowf("execute git command got error", "%s", err)
	}

	assert.Equal("refs/heads/master\n", stdout)
}
