package flickgo

import (
  "bytes"
  "crypto/md5"
  "errors"
  "fmt"
  "github.com/op/go-logging"
  "hash"
  "io"
  "io/ioutil"
  "net/http"
  "net/url"
  "strconv"
  "strings"
  "testing"
)

var log = logging.MustGetLogger("com.github.octplane.flickgo")

func init() {
  var format = logging.MustStringFormatter("%{level} %{message}")
  logging.SetFormatter(format)
  logging.SetLevel(logging.INFO, "com.github.octplane.flickgo")
}

const (
  apiKey = "87337fd784"
  secret = "sf97838dijd"
)

func assert(t *testing.T, tag string, cond bool) {
  if !cond {
    t.Errorf("[%s] assertion failed", tag)
  }
}

func assertOK(t *testing.T, id string, err error) {
  if err != nil {
    t.Errorf("[%s] unexpected error: %v", id, err)
  }
}

func assertEq(t *testing.T, id string, expected interface{}, actual interface{}) {
  if expected != actual {
    t.Errorf("[%s] expected: <%v>, found <%v>", id, expected, actual)
  }
}

func assertValuesEq(t *testing.T, id string, expected url.Values, actual url.Values) {
  assertEq(t, id+".len", len(expected), len(actual))
  for k, v := range expected {
    assertEq(t, id+".item."+k+".len", len(v), len(actual[k]))
    for i, vv := range v {
      assertEq(t, id+".item."+k+"."+strconv.Itoa(i), vv, actual[k][i])
    }
  }
}

//-----------------------
// Tests for request.go
//
func write(h hash.Hash, s string) {
  h.Write([]byte(s))
}

func TestSignedURL(t *testing.T) {
  m := md5.New()
  write(m, secret)
  write(m, "123"+"98765")
  write(m, "abc"+"abc def")
  write(m, "api_key"+"apap983 key")
  write(m, "xyz"+"xyz")
  sig := fmt.Sprintf("%x", m.Sum(nil))

  args := map[string]string{
    "abc": "abc def",
    "xyz": "xyz",
    "123": "98765",
  }
  qry := make(url.Values)
  qry.Add("abc", "abc def")
  qry.Add("xyz", "xyz")
  qry.Add("123", "98765")
  qry.Add("api_key", "apap983 key")
  qry.Add("api_sig", sig)

  expected, _ := url.Parse("https://api.flickr.com/services/srv/?" + qry.Encode())
  actual, err := url.Parse(signedURL(secret, "apap983 key", "srv", args))
  assertOK(t, "urlParse", err)
  assertEq(t, "urlScheme", expected.Scheme, actual.Scheme)
  assertEq(t, "urlHost", expected.Host, actual.Host)
  assertEq(t, "urlPath", expected.Path, actual.Path)
  assertEq(t, "urlFragment", expected.Fragment, actual.Fragment)
  assertValuesEq(t, "urlQuery", expected.Query(), actual.Query())
}

type fakeBody struct {
  error   error
  strData string
  Reader  *strings.Reader
}

func bodyWithString(source string) fakeBody {
  f := fakeBody{}
  f.strData = source
  f.Reader = strings.NewReader(f.strData)
  return f
}

func (f fakeBody) Read(buf []byte) (int, error) {
  return f.Reader.Read(buf)
}

func (f fakeBody) Close() error {
  return nil
}

type fakeRoundTripper struct {
  err   error
  body  fakeBody
  getFn func(r *http.Request) (*http.Response, error)
}

func (f fakeRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
  return f.getFn(r)
}

func newHTTPClient(getFn func(*http.Request) (*http.Response, error)) *http.Client {
  rt := fakeRoundTripper{getFn: getFn}
  return &http.Client{Transport: rt}
}

func TestFetchHttpGetFails(t *testing.T) {
  url_ := "http://some.url/?arg=value"
  err := errors.New("random error")
  getFn := func(r *http.Request) (*http.Response, error) {
    assertEq(t, "url", url_, r.URL.String())
    return nil, err
  }
  c := New(apiKey, secret, newHTTPClient(getFn))

  resp, e := fetch(c, url_)
  assert(t, "resp", resp == nil)
  assertEq(t, "err", fmt.Sprintf("GET failed: Get %s: %s", url_, err), e.Error())
}

func TestFetchSuccess(t *testing.T) {
  url_ := "http://some.url/?arg=value"
  expectedData := "response from Flickr"
  body := bodyWithString(expectedData)
  resp := http.Response{Body: body}
  getFn := func(r *http.Request) (*http.Response, error) {
    assertEq(t, "url", url_, r.URL.String())
    return &resp, nil
  }
  c := New(apiKey, secret, newHTTPClient(getFn))

  in, e := fetch(c, url_)
  assertOK(t, "fetch", e)
  buf := bytes.NewBuffer(nil)
  _, cErr := io.Copy(buf, in)
  assertOK(t, "copy", cErr)
  assert(t, "data", bytes.Equal([]byte(expectedData), buf.Bytes()))
}

func TestUploadRequest(t *testing.T) {
  data := []byte("123456\n78910\nasdfoiu\nasdfeejh")
  filename := "kitten.JPEG"
  args := map[string]string{
    "title":       "kitten",
    "description": "my cute kitten",
  }
  authToken := "ase878723623"
  c := New(apiKey, secret, nil)
  c.AuthToken = authToken
  req, rqErr := uploadRequest(c, filename, data, args)
  assertOK(t, "uploadRequest", rqErr)
  pErr := req.ParseMultipartForm(128)
  assertOK(t, "parseForm", pErr)

  args["api_key"] = apiKey
  args["auth_token"] = authToken
  args["async"] = "1"
  apiSig := sign(secret, args)

  form := req.MultipartForm
  verify := func(key, value string) {
    assertEq(t, key+" len", 1, len(form.Value[key]))
    assertEq(t, key, value, form.Value[key][0])
  }
  assertEq(t, "value len", 6, len(form.Value))
  verify("title", "kitten")
  verify("description", "my cute kitten")
  verify("api_key", apiKey)
  verify("auth_token", authToken)
  verify("api_sig", apiSig)
  verify("async", "1")

  assertEq(t, "file len", 1, len(form.File))
  assertEq(t, "photo len", 1, len(form.File["photo"]))
  assertEq(t, "filename", filename, form.File["photo"][0].Filename)
  assertEq(t, "filetype", "image/jpeg",
    form.File["photo"][0].Header.Get("Content-Type"))
  file, oErr := form.File["photo"][0].Open()
  assertOK(t, "file open", oErr)
  actual, rdErr := ioutil.ReadAll(file)
  assertOK(t, "read file", rdErr)
  assertEq(t, "photo", string(data), string(actual))
}

//-----------------------
// Tests for flickr.go
//
func TestAuthURL(t *testing.T) {
  c := New(apiKey, secret, nil)

  u, uErr := url.Parse(c.AuthURL(ReadPerm))
  assertOK(t, "parseURL", uErr)
  args, qErr := url.ParseQuery(u.RawQuery)
  assertOK(t, "parseQuery", qErr)

  for _, key := range []string{"api_key", "perms", "api_sig"} {
    if len(args[key]) != 1 {
      t.Errorf("Query argument %s has value %v", key, args[key])
    }
  }
  assertEq(t, "api_key", apiKey, args["api_key"][0])
  assertEq(t, "perms", ReadPerm, args["perms"][0])
}

func TestGetTokenURL(t *testing.T) {
  frob := "837cjnei"
  c := New(apiKey, secret, nil)

  u, uErr := url.Parse(getTokenURL(c, frob))
  assertOK(t, "parseURL", uErr)
  args, err := url.ParseQuery(u.RawQuery)
  assertOK(t, "parseQuery", err)
  assertEq(t, "method", "flickr.auth.getToken", args["method"][0])
  assertEq(t, "frob", frob, args["frob"][0])
  assertEq(t, "api_key", apiKey, args["api_key"][0])
  assertEq(t, "api_sig", 1, len(args["api_sig"]))
}

func TestGetTokenAPIFailure(t *testing.T) {
  xmlStr := `<?xml version="1.0" encoding="utf-8"?>
    <rsp stat="fail">
      <err code="97" msg="Missing signature"/>
    </rsp>`
  body := bodyWithString(xmlStr)
  resp := http.Response{Body: body}
  getFn := func(r *http.Request) (*http.Response, error) {
    return &resp, nil
  }
  c := New(apiKey, secret, newHTTPClient(getFn))
  _, _, err := c.GetToken("878243")
  assert(t, "err", err != nil)
  assert(t, "message: "+err.Error(),
    strings.Contains(err.Error(), "code 97: Missing signature"))
}

func TestGetToken(t *testing.T) {
  xmlStr := `<?xml version="1.0" encoding="utf-8"?>
    <rsp stat="ok">
      <auth>
        <token>121-84669832774</token>
        <perms>write</perms>
        <user nsid="7687633@N01" username="testuser" fullname="Test User"/>
      </auth>
    </rsp>`
  body := bodyWithString(xmlStr)

  resp := http.Response{Body: body}
  getFn := func(r *http.Request) (*http.Response, error) {
    return &resp, nil
  }
  c := New(apiKey, secret, newHTTPClient(getFn))
  tok, user, err := c.GetToken("878243")
  assertOK(t, "GetToken", err)
  assertEq(t, "token", "121-84669832774", tok)
  assertEq(t, "username", "testuser", user.UserName)
  assertEq(t, "nsid", "7687633@N01", user.NSID)
}

func TestUploadFails(t *testing.T) {
  xmlStr := `<?xml version="1.0" encoding="utf-8"?>
    <rsp stat="fail">
      <err code="5" msg="Filetype was not recognised"/>
    </rsp>`
  body := bodyWithString(xmlStr)

  resp := http.Response{Body: body}
  postFn := func(r *http.Request) (*http.Response, error) {
    return &resp, nil
  }
  c := New(apiKey, secret, newHTTPClient(postFn))
  ticket, err := c.Upload("filename", []byte("photo content"),
    map[string]string{})
  assert(t, "message: "+err.Error(),
    strings.Contains(err.Error(), "code 5: Filetype was not recognised"))
  assertEq(t, "ticket", "", ticket)
}

func TestUpload(t *testing.T) {
  xmlStr := `<?xml version="1.0" encoding="utf-8"?>
    <rsp stat="ok">
      <ticketid>363</ticketid>
    </rsp>`
  body := bodyWithString(xmlStr)
  resp := http.Response{Body: body}
  postFn := func(r *http.Request) (*http.Response, error) {
    return &resp, nil
  }
  c := New(apiKey, secret, newHTTPClient(postFn))
  ticket, err := c.Upload("filename", make([]byte, 1024*1024),
    map[string]string{})
  assertOK(t, "upload", err)
  assertEq(t, "ticket", "363", ticket)
}

func TestSearchURL(t *testing.T) {
  args := map[string]string{
    "per_page": "10",
    "user_id":  "me",
  }
  c := New(apiKey, secret, nil)

  u, uErr := url.Parse(searchURL(c, args))
  assertOK(t, "parseURL", uErr)
  a, err := url.ParseQuery(u.RawQuery)
  assertOK(t, "parseQuery", err)
  assertEq(t, "method", "flickr.photos.search", a["method"][0])
  assertEq(t, "per_page", "10", a["per_page"][0])
  assertEq(t, "user_id", "me", a["user_id"][0])
  assertEq(t, "api_key", apiKey, a["api_key"][0])
  assertEq(t, "api_sig", 1, len(a["api_sig"]))
}

func TestSearch(t *testing.T) {
  xmlStr := `<?xml version="1.0" encoding="utf-8"?>
    <rsp stat="ok">
      <photos page="1" pages="3" perpage="2" total="5">
        <photo id="1234" owner="22@N01" secret="63562" server="3" farm="1"
               title="kitten" ispublic="0" isfriend="1" isfamily="1"
							 width_t="100" height_t="100"/>
        <photo id="5678" owner="22@N01" secret="36221" server="32" farm="4"
               title="puppies" ispublic="1" isfriend="0" isfamily="0"
							 width_t="120" height_t="100"/>
      </photos>
    </rsp>`
  body := bodyWithString(xmlStr)
  resp := http.Response{Body: body}
  getFn := func(r *http.Request) (*http.Response, error) {
    return &resp, nil
  }
  c := New(apiKey, secret, newHTTPClient(getFn))
  r, err := c.Search(map[string]string{})
  assertOK(t, "search", err)
  assertEq(t, "page", "1", r.Page)
  assertEq(t, "pages", "3", r.Pages)
  assertEq(t, "perpage", "2", r.PerPage)
  assertEq(t, "total", "5", r.Total)
  assertEq(t, "len photos", 2, len(r.Photos))

  verify := func(p SearchPhoto, idx int,
    id, owner, secret, server, farm, title, isPublic, widthT, heightT string,
    ratio float64) {
    assertEq(t, fmt.Sprintf("%d.id", idx), id, p.ID)
    assertEq(t, fmt.Sprintf("%d.owner", idx), owner, p.Owner)
    assertEq(t, fmt.Sprintf("%d.secret", idx), secret, p.Secret)
    assertEq(t, fmt.Sprintf("%d.server", idx), server, p.Server)
    assertEq(t, fmt.Sprintf("%d.farm", idx), farm, p.Farm)
    assertEq(t, fmt.Sprintf("%d.title", idx), title, p.Title)
    assertEq(t, fmt.Sprintf("%d.ispublic", idx), isPublic, p.IsPublic)
    assertEq(t, fmt.Sprintf("%d.width_t", idx), widthT, p.Width_T)
    assertEq(t, fmt.Sprintf("%d.height_t", idx), heightT, p.Height_T)
    assertEq(t, fmt.Sprintf("%d.ratio", idx), ratio, p.Ratio)
  }
  verify(r.Photos[0], 0, "1234", "22@N01", "63562", "3", "1", "kitten", "0",
    "100", "100", 1.00)
  verify(r.Photos[1], 1, "5678", "22@N01", "36221", "32", "4", "puppies", "1",
    "120", "100", float64(120)/100)
}

func TestInfo(t *testing.T) {
  xmlStr := `<rsp stat="ok">
<photo id="2733" secret="123456" server="12" isfavorite="0" license="3" rotation="90" originalsecret="1bc09ce34a" originalformat="png">
  <owner nsid="12037949754@N01" username="Bees" realname="Cal Henderson" location="Bedford, UK" />
  <title>orford_castle_taster</title>
  <description>hello!</description>
  <visibility ispublic="1" isfriend="0" isfamily="0" />
  <dates posted="1100897479" taken="2004-11-19 12:51:19" takengranularity="0" lastupdate="1093022469" />
  <permissions permcomment="3" permaddmeta="2" />
  <editability cancomment="1" canaddmeta="1" />
  <comments>1</comments>
  <notes>
    <note id="313" author="12037949754@N01" authorname="Bees" x="10" y="10" w="50" h="50">foo</note>
  </notes>
  <tags>
    <tag id="1234" author="12037949754@N01" raw="woo yay">wooyay</tag>
    <tag id="1235" author="12037949754@N01" raw="hoopla">hoopla</tag>
  </tags>
  <urls>
    <url type="photopage">http://www.flickr.com/photos/bees/2733/</url>
  </urls>
</photo>
</rsp>`
  body := bodyWithString(xmlStr)
  resp := http.Response{Body: body}
  getFn := func(r *http.Request) (*http.Response, error) {
    return &resp, nil
  }
  c := New(apiKey, secret, newHTTPClient(getFn))
  c.Logger = log
  r, err := c.GetInfo("2733")

  assertOK(t, "GetInfo", err)
  assertEq(t, "license", "3", r.License)
  assertEq(t, "secret", "123456", r.Secret)
  assertEq(t, "server", "12", r.Server)
  assertEq(t, "rotation", "90", r.Rotation)
  assertEq(t, "title", "orford_castle_taster", r.Title)
  assertEq(t, "description", "hello!", r.Description)
  assert(t, "visibility.IsPublic", r.Visibility.IsPublic)
  assert(t, "visibility.IsFriend", !r.Visibility.IsFriend)
  assert(t, "visibility.IsFamily", !r.Visibility.IsFamily)
  //	"Fri, 19 Nov 2004 20:51:19 GMT"
  assertEq(t, "Dates.Posted", "1100897479", r.Dates.Posted)
  assertEq(t, "Dates.Taken", "2004-11-19 12:51:19", r.Dates.Taken)
  assertEq(t, "Dates.Lastupdate", "1093022469", r.Dates.Lastupdate)
  assertEq(t, "Dates.Takengranularity", 0, r.Dates.Takengranularity)

}

func TestGetSizes(t *testing.T) {
  xmlStr := `<rsp stat="ok">
	<sizes canblog="0" canprint="1" candownload="1">
	  <size label="Square" width="75" height="75" source="http://farm2.staticflickr.com/1103/567229075_2cf8456f01_s.jpg" url="http://www.flickr.com/photos/stewart/567229075/sizes/sq/" media="photo" />
	  <size label="Large Square" width="150" height="150" source="http://farm2.staticflickr.com/1103/567229075_2cf8456f01_q.jpg" url="http://www.flickr.com/photos/stewart/567229075/sizes/q/" media="photo" />
	  <size label="Thumbnail" width="100" height="75" source="http://farm2.staticflickr.com/1103/567229075_2cf8456f01_t.jpg" url="http://www.flickr.com/photos/stewart/567229075/sizes/t/" media="photo" />
	  <size label="Small" width="240" height="180" source="http://farm2.staticflickr.com/1103/567229075_2cf8456f01_m.jpg" url="http://www.flickr.com/photos/stewart/567229075/sizes/s/" media="photo" />
	  <size label="Small 320" width="320" height="240" source="http://farm2.staticflickr.com/1103/567229075_2cf8456f01_n.jpg" url="http://www.flickr.com/photos/stewart/567229075/sizes/n/" media="photo" />
	  <size label="Medium" width="500" height="375" source="http://farm2.staticflickr.com/1103/567229075_2cf8456f01.jpg" url="http://www.flickr.com/photos/stewart/567229075/sizes/m/" media="photo" />
	  <size label="Medium 640" width="640" height="480" source="http://farm2.staticflickr.com/1103/567229075_2cf8456f01_z.jpg?zz=1" url="http://www.flickr.com/photos/stewart/567229075/sizes/z/" media="photo" />
	  <size label="Medium 800" width="800" height="600" source="http://farm2.staticflickr.com/1103/567229075_2cf8456f01_c.jpg" url="http://www.flickr.com/photos/stewart/567229075/sizes/c/" media="photo" />
	  <size label="Large" width="1024" height="768" source="http://farm2.staticflickr.com/1103/567229075_2cf8456f01_b.jpg" url="http://www.flickr.com/photos/stewart/567229075/sizes/l/" media="photo" />
	  <size label="Original" width="2400" height="1800" source="http://farm2.staticflickr.com/1103/567229075_6dc09dc6da_o.jpg" url="http://www.flickr.com/photos/stewart/567229075/sizes/o/" media="photo" />
	</sizes>
</rsp>`
  body := bodyWithString(xmlStr)
  resp := http.Response{Body: body}
  getFn := func(r *http.Request) (*http.Response, error) {
    return &resp, nil
  }
  c := New(apiKey, secret, newHTTPClient(getFn))
  c.Logger = log
  r, err := c.GetSizes("2733")

  assertOK(t, "GetSizes", err)
  assert(t, "canblog", !r.Canblog)
  assert(t, "candownload", r.Candownload)
  assertEq(t, "size.Label", "Square", r.Sizes[0].Label)
  assertEq(t, "size.Width", 320, r.Sizes[4].Width)
  assertEq(t, "size.Height", 375, r.Sizes[5].Height)
  assertEq(t, "size.Url", "http://www.flickr.com/photos/stewart/567229075/sizes/l/", r.Sizes[8].Url)
  assertEq(t, "size.Source", "http://farm2.staticflickr.com/1103/567229075_6dc09dc6da_o.jpg", r.Sizes[9].Source)
}

func TestURL(t *testing.T) {
  p := SearchPhoto{
    Photo: Photo{ID: "id",
      Secret: "secret",
      Server: "server",
      Farm:   "fx"},
    Owner: "owner",
    Title: "title",
  }
  assertEq(t, "url", "http://farmfx.static.flickr.com/server/id_secret.jpg",
    p.URL(SizeMedium500))
  assertEq(t, "url", "http://farmfx.static.flickr.com/server/id_secret_b.jpg",
    p.URL(SizeLarge))
}

func TestCheckTicketsURL(t *testing.T) {
  tickets := []string{
    "12345",
    "23232",
    "65876",
  }
  c := New(apiKey, secret, nil)
  authToken := "ase878723623"
  c.AuthToken = authToken

  u, uErr := url.Parse(checkTicketsURL(c, tickets))
  assertOK(t, "parseURL", uErr)
  a, err := url.ParseQuery(u.RawQuery)
  assertOK(t, "parseQuery", err)
  assertEq(t, "method", "flickr.photos.upload.checkTickets", a["method"][0])
  assertEq(t, "tickets", "12345,23232,65876", a["tickets"][0])
}

func TestCheckTickets(t *testing.T) {
  xmlStr := `<?xml version="1.0" encoding="utf-8"?>
    <rsp stat="ok">
      <uploader>
        <ticket id="12345" complete="0"/>
        <ticket id="56789" complete="1" photoid="232323"/>
        <ticket id="333" invalid="1"/>
      </uploader>
    </rsp>`
  body := bodyWithString(xmlStr)
  resp := http.Response{Body: body}
  getFn := func(r *http.Request) (*http.Response, error) {
    return &resp, nil
  }
  c := New(apiKey, secret, newHTTPClient(getFn))
  statuses, err := c.CheckTickets([]string{"12345", "56789", "333"})
  assertOK(t, "checkTickets", err)
  assertEq(t, "len(statues)", 3, len(statuses))

  verify := func(status TicketStatus, idx int,
    id string, complete string, invalid string, photoid string) {
    assertEq(t, fmt.Sprintf("%d.id", idx), id, status.ID)
    assertEq(t, fmt.Sprintf("%d.complete", idx), complete, status.Complete)
    assertEq(t, fmt.Sprintf("%d.invalid", idx), invalid, status.Invalid)
    assertEq(t, fmt.Sprintf("%d.photoid", idx), photoid, status.PhotoID)
  }
  verify(statuses[0], 0, "12345", "0", "", "")
  verify(statuses[1], 1, "56789", "1", "", "232323")
  verify(statuses[2], 2, "333", "", "1", "")
}

func TestGetPhotoSetsURL(t *testing.T) {
  userID := "7687633@N01"
  c := New(apiKey, secret, nil)
  authToken := "ase878723623"
  c.AuthToken = authToken

  u, uErr := url.Parse(getPhotoSetsURL(c, userID))
  assertOK(t, "parseURL", uErr)
  a, err := url.ParseQuery(u.RawQuery)
  assertOK(t, "parseQuery", err)
  assertEq(t, "method", "flickr.photosets.getList", a["method"][0])
  assertEq(t, "user_id", userID, a["user_id"][0])
}

func TestGetSets(t *testing.T) {
  xmlStr := `<?xml version="1.0" encoding="utf-8"?>
    <rsp stat="ok">
      <photosets cancreate="1">
        <photoset id="12345" photos="35" videos="0">
          <title>Flowers</title>
          <description>All my flower pictures</description>
        </photoset>
        <photoset id="65656" photos="112" videos="32">
          <title>Sophie</title>
          <description>Photos and videos of Sophie</description>
        </photoset>
      </photosets>
    </rsp>`
  body := bodyWithString(xmlStr)
  resp := http.Response{Body: body}
  getFn := func(r *http.Request) (*http.Response, error) {
    return &resp, nil
  }
  c := New(apiKey, secret, newHTTPClient(getFn))
  sets, err := c.GetSets("me")
  assertOK(t, "getPhotoSets", err)
  assertEq(t, "len(sets)", 2, len(sets))

  verify := func(set PhotoSet, idx int,
    id, title, description string) {
    assertEq(t, fmt.Sprintf("%d.id", idx), id, set.ID)
    assertEq(t, fmt.Sprintf("%d.title", idx), title, set.Title)
    assertEq(t, fmt.Sprintf("%d.description", idx), description, set.Description)
  }
  verify(sets[0], 0, "12345", "Flowers", "All my flower pictures")
  verify(sets[1], 1, "65656", "Sophie", "Photos and videos of Sophie")
}
