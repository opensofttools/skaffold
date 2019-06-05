/*
Copyright 2019 The Skaffold Authors

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

package docker

import (
	"archive/tar"
	"context"
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDockerContext(t *testing.T) {
	for _, dir := range []string{".", "sub"} {
		t.Run(dir, func(t *testing.T) {
			tmpDir, cleanup := testutil.NewTempDir(t)
			defer cleanup()

			imageFetcher := fakeImageFetcher{}
			reset := testutil.Override(t, &RetrieveImage, imageFetcher.fetch)
			defer reset()

			artifact := &latest.DockerArtifact{
				DockerfilePath: "Dockerfile",
			}

			tmpDir.Write(dir+"/files/ignored.txt", "")
			tmpDir.Write(dir+"/files/included.txt", "")
			tmpDir.Write(dir+"/.dockerignore", "**/ignored.txt\nalsoignored.txt")
			tmpDir.Write(dir+"/Dockerfile", "FROM alpine\nCOPY ./files /files")
			tmpDir.Write(dir+"/ignored.txt", "")
			tmpDir.Write(dir+"/alsoignored.txt", "")

			resetDir := testutil.Chdir(t, tmpDir.Root())
			defer resetDir()

			reader, writer := io.Pipe()
			go func() {
				err := CreateDockerTarContext(context.Background(), writer, dir, artifact, map[string]bool{})
				if err != nil {
					writer.CloseWithError(err)
				} else {
					writer.Close()
				}
			}()

			files := make(map[string]bool)
			tr := tar.NewReader(reader)
			for {
				header, err := tr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}

				files[header.Name] = true
			}

			if files["ignored.txt"] {
				t.Error("File ignored.txt should have been excluded, but was not")
			}
			if files["alsoignored.txt"] {
				t.Error("File alsoignored.txt should have been excluded, but was not")
			}
			if files["files/ignored.txt"] {
				t.Error("File files/ignored.txt should have been excluded, but was not")
			}
			if !files["files/included.txt"] {
				t.Error("File files/included.txt should have been included, but was not")
			}
			if !files["Dockerfile"] {
				t.Error("File Dockerfile should have been included, but was not")
			}
		})
	}
}
