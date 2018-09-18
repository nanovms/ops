package main
import ("testing"
        "os"
        "fmt"
)

func TestDownloadImages(t *testing.T) {
    // remove the files to force a download
    // ignore any error from remove
    os.Remove("mkfs")
    os.Remove("staging/boot")
    os.Remove("staging/stage3")
    downloadImages()

    if _, err := os.Stat("staging/boot"); os.IsNotExist(err) {
        t.Errorf("staging/boot file not found")
    }
    
    if info, err := os.Stat("mkfs"); os.IsNotExist(err) {
        t.Errorf("mkfs not found")
    } else {
        mode := fmt.Sprintf("%04o", info.Mode().Perm())
        if mode != "0775"{
            t.Errorf("mkfs not executable")
        }
    }
    
    if _, err := os.Stat("staging/stage3"); os.IsNotExist(err) {
        t.Errorf("staging/stage3 file not found")
    }
}

// TODO 
func TestStartHypervisor(t *testing.T) {

}
