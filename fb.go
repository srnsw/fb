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
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	fb "github.com/huandu/facebook"
)

var (
	getID       = flag.Bool("i", false, "print user ID")
	getFeed     = flag.Bool("f", true, "get user feed")
	getComments = flag.Bool("c", true, "get feed comments")
	getLikes    = flag.Bool("l", false, "get users that liked posts")
	getVideos   = flag.Bool("v", true, "get video lists")
	getPhotos   = flag.Bool("p", true, "get photo lists")
	packPosts   = flag.Bool("pack", false, "build posts folders")
	packPhotos  = flag.Bool("packp", false, "build photos folders")
	dataDir     = flag.String("data", "", "location of fb data")
)

var (
	fbUser      string
	appID       string
	appSecret   string
	redirectURI string
)

func main() {
	if appID == "" {
		appID = os.Getenv("FB_APP_ID")
		appSecret = os.Getenv("FB_APP_SECRET")
		redirectURI = os.Getenv("FB_REDIRECT")
	}
	flag.Parse()
	if len(flag.Args()) > 0 {
		fbUser = flag.Arg(0)
	}
	if *packPosts || *packPhotos {
		log.Fatal(pack(*packPosts, *packPhotos, *dataDir))
	}
	if fbUser == "" {
		log.Fatal("Must provide a facebook user name or ID to harvest e.g. `fb richardlehane`")
	}
	fbUser = flag.Arg(0)
	// set config
	app := fb.New(appID, appSecret)
	app.RedirectUri = redirectURI
	tok := app.AppAccessToken()
	session := app.Session(tok)
	if *getID {
		id, _ := getUserID(session)
		fmt.Println(id)
	}
	if *getVideos {
		fmt.Println("Writing videos list...")
		writeVids(session)
	}
	if *getPhotos {
		fmt.Println("Writing photos list...")
		writePhotos(session)
		fmt.Println("Writing photos meta...")
		writePhotoMetas(session)
	}
	if *getFeed {
		fmt.Println("Writing posts lists...")
		writePIDs(session)
		fmt.Println("Writing posts...")
		writePosts(session)
	}
	os.Exit(0)
}

func writeVids(session *fb.Session) {
	srcs, err := videos(session)
	if err != nil {
		fmt.Println(err)
	} else {
		save("videos.txt", srcs)
	}
	urls := make([]string, len(srcs))
	for i, v := range srcs {
		urls[i] = strings.SplitN(v, " ", 2)[1]
	}
	save("videos_urls.txt", urls)
}

func writePhotos(session *fb.Session) {
	srcs, err := photos(session)
	if err != nil {
		fmt.Println(err)
	} else {
		save("photos.txt", srcs)
	}
	urls := make([]string, len(srcs))
	for i, v := range srcs {
		urls[i] = strings.SplitN(v, " ", 2)[0]
	}
	save("photos_urls.txt", urls)
}

func writePIDs(session *fb.Session) {
	pids, err := postIDs(session)
	if err != nil {
		fmt.Print(err)
		return
	}
	err = save("post_ids.txt", pids)
	if err != nil {
		fmt.Print(err)
	}
}

func writePhotoMetas(session *fb.Session) {
	os.Mkdir("photos", 0666)
	ret, err := load("photos.txt", 4)
	if err != nil {
		fmt.Println(err)
		return
	}
	ids := ret[1]
	for _, id := range albums {
		photo, err := photoMeta(session, v, id)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		byts, err := json.MarshalIndent(photo, "", "  ")
		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}
		ioutil.WriteFile("photos/"+photo.CreatedTime[:10]+"_"+id+".json", byts, 0666)
	}
}

func writePosts(session *fb.Session) {
	os.Mkdir("posts", 0666)
	ret, err := load("post_ids.txt", 2)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	pids, dates := ret[0], ret[1]
	for i, v := range pids {
		fmt.Printf("%d: %s\n", i, dates[i])
		post, err := post(session, v)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		byts, err := json.MarshalIndent(post, "", "  ")
		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}
		ioutil.WriteFile("posts/"+dates[i]+"_"+v+".json", byts, 0666)
	}
}

func save(name string, vals []string) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	_, err = io.WriteString(f, strings.Join(vals, "\n"))
	return err
}

func load(name string, num int) ([][]string, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	ret := make([][]string, num)
	for i := range ret {
		ret[i] = make([]string, 0, 2000)
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		str := strings.SplitN(string(scanner.Bytes()), " ", num)
		for i := range ret {
			ret[i] = append(ret[i], str[i])
		}
	}
	return ret, scanner.Err()
}

func getUserID(session *fb.Session) (string, error) {
	res, err := session.Get("/"+fbUser, nil)
	if err != nil {
		return "", err
	}
	return res.Get("id").(string), nil
}

// Just return a list of all POST IDS
func postIDs(session *fb.Session) ([]string, error) {
	posts := make([]string, 0, 2000)
	res, err := session.Get("/"+fbUser+"/feed", fb.Params{"fields": "id,created_time"})
	if err != nil {
		return nil, err
	}
	paging, err := res.Paging(session)
	if err != nil {
		return nil, err
	}
	for {
		results := paging.Data()
		for _, v := range results {
			posts = append(posts, v.Get("id").(string)+" "+v.Get("created_time").(string)[:10])
		}
		done, err := paging.Next()
		if done {
			return posts, nil
		}
		if err != nil {
			return nil, err
		}
	}
}

func photos(session *fb.Session) ([]string, error) {
	ps := make([]string, 0, 100)
	res, err := session.Get("/"+fbUser+"/photos", fb.Params{"fields": "id,album{name},link,images", "type": "uploaded", "limit": 100})
	if err != nil {
		return nil, err
	}
	paging, err := res.Paging(session)
	if err != nil {
		return nil, err
	}
	for {
		results := paging.Data()
		for _, v := range results {
			ps = append(ps, strings.Join([]string{v.Get("images.0.source").(string), v.Get("id").(string), v.Get("link").(string), v.Get("album.name").(string)}, " "))
		}
		done, err := paging.Next()
		if done {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	return ps, nil
}

func videos(session *fb.Session) ([]string, error) {
	vids := make([]string, 0, 100)
	res, err := session.Get("/"+fbUser+"/videos", fb.Params{"fields": "id,permalink_url", "limit": 100})
	if err != nil {
		return nil, err
	}
	paging, err := res.Paging(session)
	if err != nil {
		return nil, err
	}
	for {
		results := paging.Data()
		for _, v := range results {
			vids = append(vids, strings.Join([]string{v.Get("id").(string), v.Get("permalink_url").(string)}, " "))
		}
		done, err := paging.Next()
		if done {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return vids, nil
}

type From struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Photo struct {
	Id           string     `facebook:",required" json:"id"`
	CreatedTime  string     `facebook:"created_time" json:"created_time"`
	Album        string     `facebook:"-" json:"album"`
	Link         string     `facebook:"link" json:"link"`
	Label        string     `facebook:"name" json:"name"`
	LikeCount    int        `facebook:"-" json:"like_count"`
	Likes        []*Like    `facebook:"-" json:"likes,omitempty"`
	CommentCount int        `facebook:"-" json:"comment_count"`
	Comments     []*Comment `facebook:"-" json:"comments,omitempty"`
}

type Post struct {
	Id           string     `facebook:",required" json:"id"`
	CreatedTime  string     `facebook:"created_time,required" json:"created_time"`
	UpdatedTime  string     `facebook:"updated_time,required" json:"updated_time"`
	Message      string     `json:"message"`
	Type         string     `json:"type"`
	PermalinkUrl string     `facebook:"permalink_url" json:"permalink_url"`
	Link         string     `facebook:"link" json:"link"`
	Label        string     `facebook:"name" json:"name"`
	Shares       int        `facebook:"-" json:"shares"`
	LikeCount    int        `facebook:"-" json:"like_count"`
	Likes        []*Like    `facebook:"-" json:"likes,omitempty"`
	CommentCount int        `facebook:"-" json:"comment_count"`
	Comments     []*Comment `facebook:"-" json:"comments,omitempty"`
}

type Comment struct {
	Id           string     `facebook:",required" json:"id"`
	CreatedTime  string     `facebook:"created_time,required" json:"created_time"`
	From         *From      `facebook:"from" json:"from"`
	Message      string     `json:"message"`
	LikeCount    *int32     `facebook:"like_count" json:"like_count,omitempty"`
	CommentCount *int32     `facebook:"comment_count" json:"comment_count,omitempty"`
	Comments     []*Comment `facebook:"-" json:"comments,omitempty"`
}

type Like struct {
	Id   string `facebook:",required" json:"id"`
	Name string `json:"name"`
}

func totalComments(coms []*Comment) int {
	l := len(coms)
	for _, c := range coms {
		l += totalComments(c.Comments)
	}
	return l
}

func decodeComment(comment map[string]interface{}) *Comment {
	c := &Comment{
		Id:          comment["id"].(string),
		CreatedTime: comment["created_time"].(string),
		Message:     comment["message"].(string),
	}
	f := comment["from"].(map[string]interface{})
	c.From = &From{f["id"].(string), f["name"].(string)}
	return c
}

func recurseComments(session *fb.Session, id string, subcomments interface{}) ([]*Comment, error) {
	if subcomments == nil {
		return nil, nil
	}
	data := subcomments.(map[string]interface{})["data"].([]interface{})
	if len(data) > 99 {
		c, _, err := comments(session, id)
		return c, err
	}
	ret := make([]*Comment, len(data))
	for i, m := range data {
		ret[i] = decodeComment(m.(map[string]interface{}))
	}
	return ret, nil
}

func photoMeta(session *fb.Session, album string, id string) (*Photo, error) {
	res, err := session.Get("/"+id, fb.Params{"fields": "id,created_time,updated_time,link,name"})
	if err != nil {
		return nil, err
	}
	p := &Photo{}
	p.Album = album
	err = res.Decode(p)
	if err != nil {
		return nil, err
	}
	if *getComments {
		p.Comments, p.CommentCount, err = comments(session, id)
	} else {
		p.CommentCount, err = count(session, id, "comments")
	}
	if *getLikes {
		p.Likes, p.LikeCount, err = likes(session, id)
	} else {
		p.LikeCount, err = count(session, id, "likes")
	}
	return p, err
}

func post(session *fb.Session, post string) (*Post, error) {
	res, err := session.Get("/"+post, fb.Params{"fields": "id,created_time,updated_time,message,type,permalink_url,link,name,shares"})
	if err != nil {
		return nil, err
	}
	p := &Post{}
	err = res.Decode(p)
	if err != nil {
		return nil, err
	}
	p.Shares = getCount(res.Get("shares"))
	if *getComments {
		p.Comments, p.CommentCount, err = comments(session, post)
	} else {
		p.CommentCount, err = count(session, post, "comments")
	}
	if *getLikes {
		p.Likes, p.LikeCount, err = likes(session, post)
	} else {
		p.LikeCount, err = count(session, post, "likes")
	}
	return p, err
}

func count(session *fb.Session, post, path string) (int, error) {
	res, err := session.Get("/"+post+"/"+path, fb.Params{"summary": 1, "limit": 1})
	if err != nil {
		return 0, err

	}
	return getCount(res.Get("summary")), nil
}

func likes(session *fb.Session, post string) ([]*Like, int, error) {
	res, err := session.Get("/"+post+"/likes", fb.Params{"fields": "id,name", "summary": 1, "limit": 100})
	if err != nil {
		return nil, 0, err

	}
	count := getCount(res.Get("summary"))

	paging, err := res.Paging(session)
	if err != nil {
		return nil, 0, err
	}
	likes := make([]*Like, 0, count)
	for {
		results := paging.Data()
		for _, v := range results {
			like := &Like{}
			err := v.Decode(like)
			if err != nil {
				return nil, 0, err
			}
			likes = append(likes, like)
		}
		done, err := paging.Next()
		if done {
			break
		}
		if err != nil {
			return nil, 0, err
		}
	}
	return likes, count, nil
}

func getCount(summary interface{}) int {
	if summary == nil {
		return 0
	}
	sum, ok := summary.(map[string]interface{})
	if !ok {
		return 0
	}
	num, ok := sum["total_count"]
	if !ok {
		num, ok = sum["count"]
		if !ok {
			return 0
		}
	}
	c, _ := num.(json.Number).Int64()
	return int(c)
}

func comments(session *fb.Session, post string) ([]*Comment, int, error) {
	res, err := session.Get("/"+post+"/comments", fb.Params{"fields": "id,created_time,from,message,like_count,comment_count,comments.limit(100){id,created_time,from,message}", "summary": 1, "limit": 100})
	if err != nil {
		return nil, 0, err
	}

	count := getCount(res.Get("summary"))

	paging, err := res.Paging(session)
	if err != nil {
		return nil, 0, err
	}
	coms := make([]*Comment, 0, count*2)
	for {
		results := paging.Data()
		for _, v := range results {
			com := &Comment{}
			err := v.Decode(com)
			if err != nil {
				return nil, 0, err
			}
			subcomments, err := recurseComments(session, com.Id, v.Get("comments"))
			if err != nil {
				return nil, 0, err
			}
			if len(subcomments) > 0 {
				com.Comments = subcomments
			}
			coms = append(coms, com)
			if *com.CommentCount == 0 {
				com.CommentCount = nil
			}
			if *com.LikeCount == 0 {
				com.LikeCount = nil
			}
		}
		done, err := paging.Next()
		if done {
			break
		}
		if err != nil {
			return nil, 0, err
		}
	}
	return coms, count, nil
}
