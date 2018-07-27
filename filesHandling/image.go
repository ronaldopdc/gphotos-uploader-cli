package fileshandling

import (
	"fmt"
	imageLib "image"
	// register decoders for jpeg and png
	_ "image/jpeg"
	_ "image/png"

	"net/http"
	"os"
	"strings"

	"github.com/Nr90/imgsim"
	"github.com/steakknife/hamming"
	photoslibrary "google.golang.org/api/photoslibrary/v1"
)

var (
	imageExtensions = []string{".jpg", ".jpeg", ".png"}
	deletionsChan   = make(chan DeletionJob)
)

type DeletionJob struct {
	uploadedMediaItem *photoslibrary.MediaItem
	localFilePath     string
}

func QueueDeletionJob(uploadedMediaItem *photoslibrary.MediaItem, localImgPath string) {
	deletionsChan <- DeletionJob{uploadedMediaItem, localImgPath}
}

func CloseDeletionsChan() { close(deletionsChan) }

func StartDeletionsWorker() (doneDeleting chan struct{}) {
	doneDeleting = make(chan struct{})
	go func() {
		for deletionJob := range deletionsChan {
			err := CheckUploadedAndDeleteLocal(deletionJob.uploadedMediaItem, deletionJob.localFilePath)
			if err != nil {
				fmt.Printf("%s. Won't delete", err)
			}
		}
		fmt.Println("all deletions done")
		doneDeleting <- struct{}{}
	}()
	return doneDeleting
}

func imageFromPath(filePath string) (imageLib.Image, error) {
	reader, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	img, _, err := imageLib.Decode(reader)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func imageFromURL(URL string) (imageLib.Image, error) {
	res, err := http.Get(URL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("expected http status 200, got %d", res.StatusCode)
	}

	img, _, err := imageLib.Decode(res.Body)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func hasImageExtension(path string) bool {
	for _, ext := range imageExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

func isSameImage(upImg, localImg imageLib.Image) bool {
	upDHash := imgsim.DifferenceHash(upImg).String()
	localDHash := imgsim.DifferenceHash(localImg).String()

	if len(upDHash) != len(localDHash) {
		return false
	}
	hammingDistance := hamming.Strings(upDHash, localDHash)

	if hammingDistance < len(upDHash)/16 {
		return true
	}
	return false
}

// CheckUploadedAndDeleteLocal checks that the image that was uploaded is visually similar to the local one, before deleting the local one
func CheckUploadedAndDeleteLocal(uploadedMediaItem *photoslibrary.MediaItem, localImgPath string) error {
	if !hasImageExtension(localImgPath) {
		return fmt.Errorf("%s doesn't have an image extension", localImgPath)
	}

	// compare uploaded image and local one
	upImg, err := imageFromURL(uploadedMediaItem.BaseUrl)
	if err != nil {
		return fmt.Errorf("failed getting image from URL")
	}
	localImg, err := imageFromPath(localImgPath)
	if err != nil {
		return fmt.Errorf("failed loading local image from path")
	}

	if !isSameImage(upImg, localImg) {
		fmt.Println("not the same image. Won't delete")
	} else {
		fmt.Println("should delete")
		// if err = os.Remove(localImgPath); err != nil {
		// 	fmt.Println("delete failed")
		// }
	}
	return nil
}

// const imageExtensions = ["jpg", "png"].map(v => `.${v}`);

// const getImage = uri => Jimp.read(uri).then(image => image);

// const isSameImage = (upImg, localImg) => {
//   var distance = Jimp.distance(upImg, localImg); // perceived distance
//   var diff = Jimp.diff(upImg, localImg); // pixel difference
//   if (distance < 0.5 || diff.percent < 0.15) {
//     return true;
//   }
//   console.log(
//     `upImg: ${upImg.hash()}, localImg: ${localImg.hash()}, phash distance: ${distance}`
//   );
//   return false;
// };

// const deleteFile = filePath => {
//   fs.unlink(filePath, () => {
//     console.log(`DELETED ${filePath}`);
//   });
// };

// checkAndDeleteLocal = async (upImgUrl, localImgPath) => {
//   if (
//     imageExtensions.some(v =>
//       localImgPath.toLowerCase().endsWith(v.toLowerCase())
//     )
//   ) {
//     const upImgPromise = getImage(upImgUrl).then(r => r).catch(logError);
//     const localImgPromise = getImage(localImgPath).then(r => r).catch(logError);
//     Promise.all([upImgPromise, localImgPromise])
//       .then(([upImg, localImg]) => {
//         if (isSameImage(upImg, localImg)) {
//           deleteFile(localImgPath);
//         } else {
//           console.log(
//             `[ERROR] ${localImgPath}: couldn't find similar image uploaded. Won't delete.`,
//             `uploaded image at ${upImgUrl}`
//           );
//         }
//       })
//       .catch(logError);
//   } else {
//     console.log(
//       `${localImgPath} doesn't have an image extension. Won't delete`
//     );
//   }
// };
