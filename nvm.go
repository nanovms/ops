package main

import ("fmt"
        "os"
        "strings"
        "net/http"
        "os/exec"
        "io"
        "github.com/spf13/cobra"
        "path/filepath"
      )

func copy(src, dst string) error {
    in, err := os.Open(src)
    if err != nil {
        return err
    }
    defer in.Close()

    out, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer out.Close()

    _, err = io.Copy(out, in)
    if err != nil {
        return err
    }
    return out.Close()
}

func checkExists(key string) bool {
    _, err := exec.LookPath(key)
    if err != nil {
        return false
    }
    return true 
}

func startHypervisor(image string){

   // TODO: https://github.com/deferpanic/uniboot/issues/31
   err := copy(finalImg,"image2");
   panicOnError(err)
   for k := range hypervisors {
    if checkExists(k) {
      hypervisor := hypervisors[k]()
      hypervisor.start(image)
      break
    }
  }
}

func panicOnError(err error) {
  if err != nil {
    panic(err)
  }
}

func  runCommandHandler(cmd *cobra.Command, args[] string) {
   //  prepare manifest file

   var elfname = filepath.Base(args[0])
   var extension = filepath.Ext(elfname)
   elfname = elfname[0:len(elfname)-len(extension)]
   elfmanifest := fmt.Sprintf(manifest, kernelImg , args[0], elfname)
   fmt.Println(elfmanifest)
  
   // invoke mkfs to create the filesystem ie kernel + elf image
   mkfs := exec.Command("./mkfs", mergedImg)
   stdin, err := mkfs.StdinPipe()
   panicOnError(err)
   
   _, err = io.WriteString(stdin, elfmanifest)
   panicOnError(err)

   out, err := mkfs.CombinedOutput()
   if err != nil {
     fmt.Printf("%s\n", out)
     panic(err)
   }
   // produce final image, boot + kernel + elf
   fd, err := os.Create(finalImg)
   defer fd.Close()
   panicOnError(err)
   
   catcmd := exec.Command("cat", bootImg, mergedImg)
   catcmd.Stdout = fd
   err = catcmd.Start();
   panicOnError(err) 
   catcmd.Wait()

   startHypervisor(finalImg)
}

type bytesWrittenCounter struct {
  total uint64
}

func (bc *bytesWrittenCounter) Write(p []byte) (int, error) {
  n := len(p)
  bc.total += uint64(n)
  bc.printProgress()
  return n, nil
}

func (wc bytesWrittenCounter) printProgress() {
  // clear the previous line
  fmt.Printf("\r%s", strings.Repeat(" ", 35))
  fmt.Printf("\rDownloading... %v complete", wc.total)
}

func downloadFile(filepath string, url string) error {
  out, err := os.Create(filepath + ".tmp")
  if err != nil {
    return err
  }
  defer out.Close()
  resp, err := http.Get(url)
  if err != nil {
    return err
  }
  defer resp.Body.Close()
  // progress reporter.
  counter := &bytesWrittenCounter{}
  _, err = io.Copy(out, io.TeeReader(resp.Body, counter))
  if err != nil {
    return err
  }
  err = os.Rename(filepath+".tmp", filepath)
  if err != nil {
    return err
  }
  return nil
}

func downloadImages() {
  var err error
  if _, err := os.Stat("staging"); os.IsNotExist(err) {
    os.MkdirAll("staging", 0755)
  }

  if _, err = os.Stat("./mkfs"); os.IsNotExist(err) {
    err = downloadFile("mkfs",fmt.Sprintf(bucketBaseUrl, "mkfs"))
    panicOnError(err)
  }

  // make mkfs executable
  err = os.Chmod("mkfs",0755)
  if err != nil {
      panicOnError(err)
  }

  if _, err = os.Stat("staging/boot"); os.IsNotExist(err) {
    err = downloadFile("staging/boot",fmt.Sprintf(bucketBaseUrl, "boot"))
    panicOnError(err)
  }

  if _, err = os.Stat("staging/stage3"); os.IsNotExist(err) {
    err = downloadFile("staging/stage3",fmt.Sprintf(bucketBaseUrl, "stage3"))
    panicOnError(err)
  }

}

func main(){
    var cmdPrint = &cobra.Command {
        Use:   "run [ELF file]",
        Short: "run ELF as unikernel",
        Args: cobra.MinimumNArgs(1),
        Run: runCommandHandler,
    }
  var rootCmd = &cobra.Command{Use: "nvm"}
  rootCmd.AddCommand(cmdPrint)
  downloadImages()
  rootCmd.Execute()
}
