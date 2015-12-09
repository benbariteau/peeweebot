package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v2"
)

var configDir = flag.String("config", "~/.peeweebot/", "location of config directory")

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

func main() {
	ctx := context.Background()
	driveService, err := getGoogleDriveService(ctx)
	if err != nil {
		log.Fatalf("Unable to get google drive service: %v", driveService)
	}

	r, err := driveService.Files.List().MaxResults(10).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve files.", err)
	}

	fmt.Println("Files:")
	if len(r.Items) > 0 {
		for _, i := range r.Items {
			fmt.Printf("%s (%s)\n", i.Title, i.Id)
		}
	} else {
		fmt.Print("No files found.")
	}
}
