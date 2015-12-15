package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/ChimeraCoder/anaconda"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v2"
)

var configDir = flag.String("config", "~/.peeweebot/", "location of config directory")
var folderId = "0B1SaB_OdyoZrVEhQR01WWXoxbjA"

func getConfigDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("Unable to get current user: %v", err)
	}
	return filepath.Clean(strings.Replace(*configDir, "~/", usr.HomeDir+"/", 1))
}

func getGoogleOAuthConfig() *oauth2.Config {
	b, err := ioutil.ReadFile(filepath.Join(getConfigDir(), "google_client_secrets.json"))
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, drive.DriveReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	return config
}

func getGoogleDriveTokenFromFile() *oauth2.Token {
	filename := filepath.Join(getConfigDir(), "google_drive_oauth_token.json")
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Unable to open token file \"%v\": %v", filename, err)
	}
	defer f.Close()

	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	if err != nil {
		log.Fatalf("Unable to decode token file \"%v\": %v", filename, err)
	}

	return t
}

func getGoogleDriveService(ctx context.Context) (*drive.Service, error) {
	return drive.New(
		getGoogleOAuthConfig().Client(
			ctx,
			getGoogleDriveTokenFromFile(),
		),
	)
}

func getAllChildren(driveService *drive.Service, folder string) (list []*drive.ChildReference) {
	var pageToken string
	for {
		call := driveService.Children.List(folder)
		if pageToken != "" {
			call.PageToken(pageToken)
		}

		r, err := call.Do()
		if err != nil {
			log.Fatalf("Unable to retrieve files.", err)
		}

		list = append(list, r.Items...)

		pageToken = r.NextPageToken

		if pageToken == "" {
			break
		}
	}
	return
}

type TwitterStuff struct {
	ConsumerKey       string
	ConsumerSecret    string
	AccessToken       string
	AccessTokenSecret string
}

func getTwitterClient() *anaconda.TwitterApi {
	fd, err := os.Open(filepath.Join(getConfigDir(), "twitter_stuff.json"))
	if err != nil {
		log.Fatalf("Unable to open twitter_stuff.json: %v", err)
	}
	defer fd.Close()

	twitterStuff := TwitterStuff{}
	err = json.NewDecoder(fd).Decode(&twitterStuff)
	if err != nil {
		log.Fatalf("Unable to decode twitter_stuff.json: %v", err)
	}

	anaconda.SetConsumerKey(twitterStuff.ConsumerKey)
	anaconda.SetConsumerSecret(twitterStuff.ConsumerSecret)
	return anaconda.NewTwitterApi(
		twitterStuff.AccessToken,
		twitterStuff.AccessTokenSecret,
	)
}

func tweetRandomPicture(driveService *drive.Service, twitterApi *anaconda.TwitterApi) {
	fileList := getAllChildren(driveService, folderId)

	fileNumber := rand.Intn(len(fileList))
	selectedFile := fileList[fileNumber]

	fileResponse, err := driveService.Files.Get(selectedFile.Id).Download()
	if err != nil {
		log.Fatalf("Unable to fetch file: %v", err)
	}
	defer fileResponse.Body.Close()

	buffer := new(bytes.Buffer)
	writer := base64.NewEncoder(base64.StdEncoding, buffer)

	_, err = io.Copy(writer, fileResponse.Body)
	if err != nil {
		log.Fatalf("Unable to encode image as base64: %v", err)
	}

	media, err := twitterApi.UploadMedia(buffer.String())
	if err != nil {
		log.Fatalf("Unable to upload media: %v", err)
	}

	params := url.Values{}
	params.Set("media_ids", media.MediaIDString)

	_, err = twitterApi.PostTweet("", params)
	if err != nil {
		log.Fatalf("Unable to post tweet: %v", err)
	}
}

func main() {
	rand.Seed(time.Now().Unix())
	ctx := context.Background()
	driveService, err := getGoogleDriveService(ctx)
	if err != nil {
		log.Fatalf("Unable to get google drive service: %v", err)
	}
	twitterApi := getTwitterClient()

	tweetRandomPicture(driveService, twitterApi)
}
