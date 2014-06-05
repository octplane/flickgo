// Copyright 2011 Muthukannan T <manki@manki.in>. All Rights Reserved.

package flickgo

import (
	"fmt"
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
}

type Visibility struct {
	IsPublic bool `xml:"ispublic"`
	IsFriend bool `xml:"isfriend"`
	IsFamily bool `xml:"isfamily"`
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
