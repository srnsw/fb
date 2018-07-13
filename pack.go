// Copyright 2018 State of New South Wales through the State Archives and Records Authority of NSW
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/srnsw/wincommands"
)

type pathToID func(string) string

func videoID(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	sep := strings.LastIndex(base, "-")
	if sep > 0 {
		return base[sep+1:]
	}
	return ""
}

func photoID(path string) string {
	base := filepath.Base(path)
	bits := strings.Split(base, "_")
	if len(bits) >= 2 {
		return bits[1]
	}
	return ""
}

func addLinks(m map[string]string, dir, path string) (map[string]string, error) {
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		bits := strings.SplitN(scan.Text(), " ", 4)
		id := photoID(bits[0])
		if p, ok := m[id]; ok {
			m[bits[2]] = p
		} else {
			sep := strings.LastIndex(bits[0], "/")
			m[bits[2]] = filepath.Join(dir, bits[0][sep+1:])
		}
	}
	return m, nil
}

func buildMap(path string, idfunc pathToID) (map[string]string, error) {
	ret := make(map[string]string)
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		id := idfunc(path)
		if id != "" {
			if _, ok := ret[id]; ok {
				panic(fmt.Errorf("Double-up for id %s, path %s", id, path))
			}
			ret[id] = path
		}
		return nil
	})
	return ret, nil
}

func simpleID(id string) string {
	split := strings.SplitN(id, "_", 2)
	return split[len(split)-1]
}

func obj(photosMap, videosMap map[string]string, dir string, g gen, r rdr, w io.Writer) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) != ".json" {
			return nil
		}
		byt, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		entity := g()
		if err = json.Unmarshal(byt, entity); err != nil {
			return err
		}
		id, create, link := r(entity)
		target := filepath.Join(dir, "out", create+"_"+id)
		photo, pok := photosMap[id]
		if pok {
			if err = wincommands.FileCopyLog(w, photo, target, false); err != nil {
				return err
			}
		} else {
			photo, pok = photosMap[link]
			if pok {
				if err = wincommands.FileCopyLog(w, photo, target, false); err != nil {
					return err
				}
			}
		}
		video, vok := videosMap[id]
		if vok {
			if err = wincommands.FileCopyLog(w, video, target, false); err != nil {
				return err
			}
		}
		return wincommands.FileCopy(path, target, false)
	}
}

type gen func() interface{}

func postGen() interface{} {
	return &Post{}
}

func photoGen() interface{} {
	return &Photo{}
}

type rdr func(v interface{}) (id, created, link string)

func postRdr(v interface{}) (string, string, string) {
	post := v.(*Post)
	return simpleID(post.Id), post.CreatedTime[:10], post.Link
}

func photoRdr(v interface{}) (string, string, string) {
	photo := v.(*Photo)
	return simpleID(photo.Id), photo.CreatedTime[:10], photo.Link
}

func pack(postsf, photosf bool, dir string) error {
	photosMap, err := buildMap(filepath.Join(dir, "media", "photos"), photoID)
	if err != nil {
		return err
	}
	photosMap, err = addLinks(photosMap, filepath.Join(dir, "media", "photos"), filepath.Join(dir, "photos.txt"))
	videosMap, err := buildMap(filepath.Join(dir, "media", "videos"), videoID)
	if err != nil {
		return err
	}
	run := filepath.Join(dir, "run.bat")
	_ = os.Remove(run) // clear previous runs if any
	lg, err := os.Create(run)
	defer lg.Close()
	if err != nil {
		return err
	}
	if postsf {
		if err = filepath.Walk(filepath.Join(dir, "posts"),
			obj(photosMap, videosMap, dir, postGen, postRdr, lg)); err != nil {
			return err
		}
	}
	if photosf {
		if err = filepath.Walk(filepath.Join(dir, "photos"),
			obj(photosMap, videosMap, dir, photoGen, photoRdr, lg)); err != nil {
			return err
		}
	}
	return nil
}
