package main

import (
	"bufio"
	"encoding/json"
	"fmt"
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

func obj(photosMap, videosMap map[string]string, linked map[string]bool, dir string, g gen, r rdr) filepath.WalkFunc {
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
		id, create, link, typ := r(entity)
		target := filepath.Join(dir, "out", create+"_"+id)
		photo, pok := photosMap[id]
		if pok {
			if err = wincommands.FileCopyList(photo, target, false); err != nil {
				return err
			}
		} else {
			photo, pok = photosMap[link]
			if err = wincommands.FileCopyList(photo, target, false); err != nil {
				return err
			}
		}
		video, vok := videosMap[id]
		if vok {
			if err = wincommands.FileCopyList(video, target, false); err != nil {
				return err
			}
		}
		if !vok && !pok {
			switch typ {
			case "link", "status":
			default:
				fmt.Println(id + " " + typ)
			}
		} else {
			if pok {
				linked[photo] = true
			} else {
				linked[video] = true
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

type rdr func(v interface{}) (id, created, link, typ string)

func postRdr(v interface{}) (string, string, string, string) {
	post := v.(*Post)
	return simpleID(post.Id), post.CreatedTime[:10], post.Link, post.Type
}

func photoRdr(v interface{}) (string, string, string, string) {
	photo := v.(*Photo)
	return simpleID(photo.Id), photo.CreatedTime[:10], photo.Link, "non post photo"
}

func pack(postsf, photosf bool, dir string) error {
	linked := make(map[string]bool)
	photosMap, err := buildMap(filepath.Join(dir, "media", "photos"), photoID)
	if err != nil {
		return err
	}
	photosMap, err = addLinks(photosMap, filepath.Join(dir, "media", "photos"), filepath.Join(dir, "photos.txt"))
	videosMap, err := buildMap(filepath.Join(dir, "media", "videos"), videoID)
	if err != nil {
		return err
	}
	if postsf {
		if err = filepath.Walk(filepath.Join(dir, "posts"),
			obj(photosMap, videosMap, linked, dir, postGen, postRdr)); err != nil {
			return err
		}
	}
	if photosf {
		if err = filepath.Walk(filepath.Join(dir, "photos"),
			obj(photosMap, videosMap, linked, dir, photoGen, photoRdr)); err != nil {
			return err
		}
	}
	missing := make(map[string]bool)
	for _, v := range photosMap {
		if !linked[v] {
			missing[v] = true
		}
	}
	for _, v := range videosMap {
		if !linked[v] {
			missing[v] = true
		}
	}
	for k, _ := range missing {
		fmt.Println(k)
	}
	return nil
}
