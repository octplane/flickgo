// Copyright 2011 Muthukannan T <manki@manki.in>. All Rights Reserved.

package flickgo

import (
  "fmt"
  "strconv"
  "time"
)

// Image sizes supported by Flickr.  See
// http://www.flickr.com/services/api/misc.urls.html for more information.
const (
  SizeSmallSquare = "s"
  SizeThumbnail   = "t"
  SizeSmall       = "m"
  SizeMedium500   = "-"
  SizeMedium640   = "z"
  SizeLarge       = "b"
  SizeOriginal    = "o"
)

// Response for photo search requests.
type SearchResponse struct {
  Page    string  `xml:"page,attr"`
  Pages   string  `xml:"pages,attr"`
  PerPage string  `xml:"perpage,attr"`
  Total   string  `xml:"total,attr"`
  Photos  []Photo `xml:"photo"`
}

type InfoResponse struct {
  ID          string     `xml:"id,attr"`
  Secret      string     `xml:"secret,attr"`
  Server      string     `xml:"server,attr"`
  Rotation    string     `xml:"rotation,attr"`
  License     string     `xml:"license,attr"`
  Title       string     `xml:"title"`
  Description string     `xml:"description"`
  Visibility  Visibility `xml:"visibility"`
  Dates       Dates      `xml:"dates"`
  Tags        []Tag      `xml:"tags>tag"`
  Urls        []Url      `xml:"urls>url"`
}

type SizesResponse struct {
  Canblog     bool   `xml:"canblog,attr"`
  Canprint    bool   `xml:"canprint,attr"`
  Candownload bool   `xml:"candownload,attr"`
  Sizes       []Size `xml:"size"`
}

type Size struct {
  Label  string `xml:"label,attr"`
  Width  int    `xml:"width,attr"`
  Height int    `xml:"height,attr"`
  Source string `xml:"source,attr"`
  Url    string `xml:"url,attr"`
}

type Visibility struct {
  IsPublic bool `xml:"ispublic,attr"`
  IsFriend bool `xml:"isfriend,attr"`
  IsFamily bool `xml:"isfamily,attr"`
}

type Dates struct {
  Posted           string `xml:"posted,attr"` // Unix timestamp
  Taken            string `xml:"taken,attr"`
  Takengranularity int    `xml:"takengranularity,attr"`
  Lastupdate       string `xml:"lastupdate,attr"`
}

type Tag struct {
  ID   string `xml:"id,attr"`
  Text string `xml:",chardata"`
}

type Url struct {
  Type string `xml:"type,attr"`
  Href string `xml:",chardata"`
}

func stringToTime(source string) time.Time {
  pd, err := strconv.ParseInt(source, 10, 64)
  if err != nil {
    panic(err)
  }
  return time.Unix(pd, 0)
}

func (d *Dates) PostedTime() time.Time {
  return stringToTime(d.Posted)
}

func (d *Dates) TakenTime() time.Time {
  return stringToTime(d.Taken)
}

func (d *Dates) LastupdateTime() time.Time {
  return stringToTime(d.Lastupdate)
}

// A Flickr user.
type User struct {
  UserName string `xml:"username,attr"`
  NSID     string `xml:"nsid,attr"`
}

// Represents a Flickr photo.
type Photo struct {
  ID       string `xml:"id,attr"`
  Owner    string `xml:"owner,attr"`
  Secret   string `xml:"secret,attr"`
  Server   string `xml:"server,attr"`
  Farm     string `xml:"farm,attr"`
  Title    string `xml:"title,attr"`
  IsPublic string `xml:"ispublic,attr"`
  Width_T  string `xml:"width_t,attr"`
  Height_T string `xml:"height_t,attr"`
  // Photo's aspect ratio: width divided by height.
  Ratio float64
}

// Returns the URL to this photo in the specified size.
func (p *Photo) URL(size string) string {
  if size == "-" {
    return fmt.Sprintf("http://farm%s.static.flickr.com/%s/%s_%s.jpg",
      p.Farm, p.Server, p.ID, p.Secret)
  }
  return fmt.Sprintf("http://farm%s.static.flickr.com/%s/%s_%s_%s.jpg",
    p.Farm, p.Server, p.ID, p.Secret, size)
}

type PhotoSet struct {
  ID          string `xml:"id,attr"`
  Title       string `xml:"title"`
  Description string `xml:"description"`
}
