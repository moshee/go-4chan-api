// Package api pulls 4chan board and thread data from the JSON API into native Go data structures.
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	pathpkg "path"
	"sync"
	"time"
)

var (
	// Whether or not to use HTTPS for requests.
	SSL bool = false
	// Cooldown time for updating threads using (*Thread).Update().
	// If it is set to less than 10 seconds, it will be re-set to 10 seconds
	// before being used.
	UpdateCooldown time.Duration = 15 * time.Second
	cooldown       <-chan time.Time
	cooldownMutex  sync.Mutex
)

const (
	APIURL    = "a.4cdn.org"
	ImageURL  = "i.4cdn.org"
	StaticURL = "s.4cdn.org"
)

func prefix() string {
	if SSL {
		return "https://"
	} else {
		return "http://"
	}
}

func get(base, path string, modify func(*http.Request) error) (*http.Response, error) {
	url := prefix() + pathpkg.Join(base, path)
	cooldownMutex.Lock()
	if cooldown != nil {
		<-cooldown
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if modify != nil {
		err = modify(req)
		if err != nil {
			return nil, err
		}
	}

	resp, err := http.DefaultClient.Do(req)
	cooldown = time.After(1 * time.Second)
	cooldownMutex.Unlock()
	return resp, err
}

func getDecode(base, path string, dest interface{}, modify func(*http.Request) error) error {
	resp, err := get(base, path, modify)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(dest)
}

// Direct mapping from the API's JSON to a Go type.
type jsonPost struct {
	No             int64            `json:"no"`             // Post number         1-9999999999999
	Resto          int64            `json:"resto"`          // Reply to            0 (is thread), 1-999999999999
	Sticky         int              `json:"sticky"`         // Stickied thread?    0 (no), 1 (yes)
	Closed         int              `json:"closed"`         // Closed thread?      0 (no), 1 (yes)
	Now            string           `json:"now"`            // Date and time       MM\/DD\/YY(Day)HH:MM (:SS on some boards)
	Time           int64            `json:"time"`           // UNIX timestamp      UNIX timestamp
	Name           string           `json:"name"`           // Name                text or empty
	Trip           string           `json:"trip"`           // Tripcode            text (format: !tripcode!!securetripcode)
	Id             string           `json:"id"`             // ID                  text (8 characters), Mod, Admin
	Capcode        string           `json:"capcode"`        // Capcode             none, mod, admin, admin_highlight, developer
	Country        string           `json:"country"`        // Country code        ISO 3166-1 alpha-2, XX (unknown)
	CountryName    string           `json:"country_name"`   // Country name        text
	Email          string           `json:"email"`          // Email               text or empty
	Sub            string           `json:"sub"`            // Subject             text or empty
	Com            string           `json:"com"`            // Comment             text (includes escaped HTML) or empty
	Tim            int64            `json:"tim"`            // Renamed filename    UNIX timestamp + microseconds
	FileName       string           `json:"filename"`       // Original filename   text
	Ext            string           `json:"ext"`            // File extension      .jpg, .png, .gif, .pdf, .swf
	Fsize          int              `json:"fsize"`          // File size           1-8388608
	Md5            []byte           `json:"md5"`            // File MD5            byte slice
	Width          int              `json:"w"`              // Image width         1-10000
	Height         int              `json:"h"`              // Image height        1-10000
	TnW            int              `json:"tn_w"`           // Thumbnail width     1-250
	TnH            int              `json:"tn_h"`           // Thumbnail height    1-250
	FileDeleted    int              `json:"filedeleted"`    // File deleted?       0 (no), 1 (yes)
	Spoiler        int              `json:"spoiler"`        // Spoiler image?      0 (no), 1 (yes)
	CustomSpoiler  int              `json:"custom_spoiler"` // Custom spoilers?	1-99
	OmittedPosts   int              `json:"omitted_posts"`  // # replies omitted	1-10000
	OmittedImages  int              `json:"omitted_images"` // # images omitted	1-10000
	Replies        int              `json:"replies"`        // total # of replies	0-99999
	Images         int              `json:"images"`         // total # of images	0-99999
	BumpLimit      int              `json:"bumplimit"`      // bump limit?			0 (no), 1 (yes)
	ImageLimit     int              `json:"imagelimit"`     // image limit?		0 (no), 1 (yes)
	CapcodeReplies map[string][]int `json:"capcode_replies"`
	LastModified   int64            `json:"last_modified"`
}

// A Post represents all of the attributes of a 4chan post, organized in a more directly usable fashion.
type Post struct {
	// Post info
	Id           int64
	Thread       *Thread
	Time         time.Time
	Subject      string
	LastModified int64

	// These are only present in an OP post. They are exposed through their
	// corresponding Thread getter methods.
	replies        int
	images         int
	omitted_posts  int
	omitted_images int
	bump_limit     bool
	image_limit    bool
	sticky         bool
	closed         bool
	custom_spoiler int // the number of custom spoilers on a given board

	// Poster info
	Name    string
	Trip    string
	Email   string
	Special string
	Capcode string

	// Country and CountryName are empty unless the board uses country info
	Country     string
	CountryName string

	// Message body
	Comment string

	// File info if any, otherwise nil
	File *File

	// only when they do this on /q/
	CapcodeReplies map[string][]int
}

func (self *Post) String() (s string) {
	s += fmt.Sprintf("#%d %s%s on %s:\n", self.Id, self.Name, self.Trip, self.Time.Format(time.RFC822))
	if self.File != nil {
		s += self.File.String()
	}
	s += self.Comment
	return
}

// ImageURL constructs and returns the URL of the attached image. Returns the
// empty string if there is none.
func (self *Post) ImageURL() string {
	file := self.File
	if file == nil {
		return ""
	}
	return fmt.Sprintf("%s%s/%s/%d%s",
		prefix(), ImageURL, self.Thread.Board, file.Id, file.Ext)
}

// ThumbURL constructs and returns the thumbnail URL of the attached image.
// Returns the empty string if there is none.
func (self *Post) ThumbURL() string {
	file := self.File
	if file == nil {
		return ""
	}
	return fmt.Sprintf("%s%s/%s/%ds%s",
		prefix(), ImageURL, self.Thread.Board, file.Id, ".jpg")
}

// A File represents an uploaded file's metadata.
type File struct {
	Id          int64  // Id is what 4chan renames images to (UNIX + microtime, e.g. 1346971121077)
	Name        string // Original filename
	Ext         string
	Size        int
	MD5         []byte
	Width       int
	Height      int
	ThumbWidth  int
	ThumbHeight int
	Deleted     bool
	Spoiler     bool
}

func (self *File) String() string {
	return fmt.Sprintf("File: %s%s (%dx%d, %d bytes, md5 %x)\n",
		self.Name, self.Ext, self.Width, self.Height, self.Size, self.MD5)
}

// CountryFlagURL returns the URL of the post's country flag icon, if enabled
// on the board in question.
func (self *Post) CountryFlagURL() string {
	if self.Country == "" {
		return ""
	}
	// lol /pol/
	if self.Thread.Board == "pol" {
		return fmt.Sprintf("%s://%s/image/country/troll/%s.gif", prefix(), StaticURL, self.Country)
	}
	return fmt.Sprintf("%s://%s/image/country/%s.gif", prefix(), StaticURL, self.Country)
}

// A Thread represents a thread of posts. It may or may not contain the actual replies.
type Thread struct {
	Posts []*Post
	OP    *Post
	Board string // without slashes ex. "g" or "ic"

	date_recieved time.Time
	cooldown      <-chan time.Time
}

// GetIndex hits the API for an index of thread stubs from the given board and
// page.
func GetIndex(board string, page int) ([]*Thread, error) {
	resp, err := get(APIURL, fmt.Sprintf("/%s/%d.json", board, page), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	threads, err := ParseIndex(resp.Body, board)
	now := time.Now()
	for _, t := range threads {
		t.date_recieved = now
	}
	return threads, err
}

// GetThreads hits the API for a list of the thread IDs of all the active
// threads on a given board.
func GetThreads(board string) ([][]int64, error) {
	p := make([]struct {
		Page    int `json:"page"`
		Threads []struct {
			No int64 `json:"no"`
		} `json:"threads"`
	}, 0, 10)
	if err := getDecode(APIURL, fmt.Sprintf("/%s/threads.json", board), &p, nil); err != nil {
		return nil, err
	}
	n := make([][]int64, len(p))
	for _, page := range p {
		n[page.Page] = make([]int64, len(page.Threads))
		for j, thread := range page.Threads {
			n[page.Page][j] = thread.No
		}
	}
	return n, nil
}

// GetThread hits the API for a single thread and all its replies. board is
// just the board name, without the surrounding slashes. If a thread is being
// updated, use an existing thread's Update() method if possible because that
// uses If-Modified-Since in the request, which reduces unnecessary server
// load.
func GetThread(board string, thread_id int64) (*Thread, error) {
	return getThread(board, thread_id, time.Unix(0, 0))
}

func getThread(board string, thread_id int64, stale_time time.Time) (*Thread, error) {
	resp, err := get(APIURL, fmt.Sprintf("/%s/thread/%d.json", board, thread_id), func(req *http.Request) error {
		if stale_time.Unix() != 0 {
			req.Header.Add("If-Modified-Since", stale_time.UTC().Format(http.TimeFormat))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	thread, err := ParseThread(resp.Body, board)
	thread.date_recieved = time.Now()

	return thread, err
}

// ParseIndex converts a JSON response for multiple threads into a native Go
// data structure
func ParseIndex(r io.Reader, board string) ([]*Thread, error) {
	var t struct {
		Threads []struct {
			Posts []*jsonPost `json:"posts"`
		} `json:"threads"`
	}

	if err := json.NewDecoder(r).Decode(&t); err != nil {
		return nil, err
	}

	threads := make([]*Thread, len(t.Threads))
	for i, json_thread := range t.Threads {
		thread := &Thread{Posts: make([]*Post, len(t.Threads[i].Posts)), Board: board}
		for k, v := range json_thread.Posts {
			thread.Posts[k] = json_to_native(v, thread)
			if v.No == 0 {
				thread.OP = thread.Posts[k]
			}
		}
		// TODO: fix this up
		if thread.OP == nil {
			thread.OP = thread.Posts[0]
		}
		threads[i] = thread
	}

	return threads, nil
}

// ParseThread converts a JSON response for one thread into a native Go data
// structure.
func ParseThread(r io.Reader, board string) (*Thread, error) {
	var t struct {
		Posts []*jsonPost `json:"posts"`
	}

	if err := json.NewDecoder(r).Decode(&t); err != nil {
		return nil, err
	}

	thread := &Thread{Posts: make([]*Post, len(t.Posts)), Board: board}
	for k, v := range t.Posts {
		thread.Posts[k] = json_to_native(v, thread)
		if v.No == 0 {
			thread.OP = thread.Posts[k]
		}
	}
	// TODO: fix this up
	if thread.OP == nil {
		thread.OP = thread.Posts[0]
	}

	return thread, nil
}

func json_to_native(v *jsonPost, thread *Thread) *Post {
	p := &Post{
		Id:             v.No,
		sticky:         v.Sticky == 1,
		closed:         v.Closed == 1,
		Time:           time.Unix(v.Time, 0),
		Name:           v.Name,
		Trip:           v.Trip,
		Special:        v.Id,
		Capcode:        v.Capcode,
		Country:        v.Country,
		CountryName:    v.CountryName,
		Email:          v.Email,
		Subject:        v.Sub,
		Comment:        v.Com,
		custom_spoiler: v.CustomSpoiler,
		replies:        v.Replies,
		images:         v.Images,
		omitted_posts:  v.OmittedPosts,
		omitted_images: v.OmittedImages,
		bump_limit:     v.BumpLimit == 1,
		image_limit:    v.ImageLimit == 1,
		Thread:         thread,
		CapcodeReplies: v.CapcodeReplies,
		LastModified:   v.LastModified,
	}
	if len(v.FileName) > 0 {
		p.File = &File{
			Id:          v.Tim,
			Name:        v.FileName,
			Ext:         v.Ext,
			Size:        v.Fsize,
			MD5:         v.Md5,
			Width:       v.Width,
			Height:      v.Height,
			ThumbWidth:  v.TnW,
			ThumbHeight: v.TnH,
			Deleted:     v.FileDeleted == 1,
			Spoiler:     v.Spoiler == 1,
		}
	}
	return p
}

// Update an existing thread in-place.
func (self *Thread) Update() (new_posts, deleted_posts int, err error) {
	cooldownMutex.Lock()
	if self.cooldown != nil {
		<-self.cooldown
	}
	var thread *Thread
	thread, err = getThread(self.Board, self.Id(), self.date_recieved)
	if UpdateCooldown < 10*time.Second {
		UpdateCooldown = 10 * time.Second
	}
	self.cooldown = time.After(UpdateCooldown)
	cooldownMutex.Unlock()
	if err != nil {
		return 0, 0, err
	}
	var a, b int
	// traverse both threads in parallel to check for deleted/appended posts
	for a, b = 0, 0; a < len(self.Posts); a, b = a+1, b+1 {
		if self.Posts[a].Id == thread.Posts[b].Id {
			continue
		}
		// a post has been deleted, go back one to compare with the next
		b--
		deleted_posts++
	}
	new_posts = len(thread.Posts) - b
	self.Posts = thread.Posts
	return
}

// Id returns the thread OP's post ID.
func (self *Thread) Id() int64 {
	return self.OP.Id
}

func (self *Thread) String() (s string) {
	for _, post := range self.Posts {
		s += post.String() + "\n\n"
	}
	return
}

// Replies returns the number of replies the thread OP has.
func (self *Thread) Replies() int {
	return self.OP.replies
}

// Images returns the number of images in the thread.
func (self *Thread) Images() int {
	return self.OP.images
}

// OmittedPosts returns the number of posts omitted in a thread list overview.
func (self *Thread) OmittedPosts() int {
	return self.OP.omitted_posts
}

// OmittedImages returns the number of image posts omitted in a thread list overview.
func (self *Thread) OmittedImages() int {
	return self.OP.omitted_images
}

// BumpLimit returns true if the thread is at its bump limit, or false otherwise.
func (self *Thread) BumpLimit() bool {
	return self.OP.bump_limit
}

// ImageLimit returns true if the thread can no longer accept image posts, or false otherwise.
func (self *Thread) ImageLimit() bool {
	return self.OP.image_limit
}

// Closed returns true if the thread is closed for replies, or false otherwise.
func (self *Thread) Closed() bool {
	return self.OP.closed
}

// Sticky returns true if the thread is stickied, or false otherwise.
func (self *Thread) Sticky() bool {
	return self.OP.sticky
}

// CustomSpoiler returns the ID of its custom spoiler image, if there is one.
func (self *Thread) CustomSpoiler() int {
	return self.OP.custom_spoiler
}

// CustomSpoilerURL builds and returns the URL of the custom spoiler image, or
// an empty string if none exists.
func (self *Thread) CustomSpoilerURL(id int, ssl bool) string {
	if id > self.OP.custom_spoiler {
		return ""
	}
	return fmt.Sprintf("%s://%s/image/spoiler-%s%d.png", prefix(), StaticURL, self.Board, id)
}

// A Board is the name and title of a single board.
type Board struct {
	Board string `json:"board"`
	Title string `json:"title"`
}

// Board names/descriptions will be cached here after a call to LookupBoard or GetBoards
var Boards []Board

// LookupBoard returns the Board corresponding to the board name (without slashes)
func LookupBoard(name string) (Board, error) {
	if Boards == nil {
		_, err := GetBoards()
		if err != nil {
			return Board{}, fmt.Errorf("Board '%s' not found: %v", name, err)
		}
	}
	for _, b := range Boards {
		if name == b.Board {
			return b, nil
		}
	}
	return Board{}, fmt.Errorf("Board '%s' not found", name)
}

// Get the list of boards.
func GetBoards() ([]Board, error) {
	var b struct {
		Boards []Board `json:"boards"`
	}
	err := getDecode(APIURL, "/boards.json", &b, nil)
	if err != nil {
		return nil, err
	}
	Boards = b.Boards
	return b.Boards, nil
}

// A Catalog contains a list of (truncated) threads on each page of a board.
type Catalog []struct {
	Page    int
	Threads []*Thread
}

type catalog []struct {
	Page    int         `json:"page"`
	Threads []*jsonPost `json:"threads"`
}

// GetCatalog hits the API for a catalog listing of a board.
func GetCatalog(board string) (Catalog, error) {
	if len(board) == 0 {
		return nil, fmt.Errorf("api: GetCatalog: No board name given")
	}
	var c catalog
	err := getDecode(APIURL, fmt.Sprintf("/%s/catalog.json", board), c, nil)
	if err != nil {
		return nil, err
	}

	cat := make(Catalog, len(c))
	for i, page := range c {
		extracted := struct {
			Page    int
			Threads []*Thread
		}{page.Page, make([]*Thread, len(page.Threads))}
		for j, post := range page.Threads {
			thread := &Thread{Posts: make([]*Post, 1), Board: board}
			post := json_to_native(post, thread)
			thread.Posts[0] = post
			extracted.Threads[j] = thread
		}
		cat[i] = extracted
	}
	return cat, nil
}
