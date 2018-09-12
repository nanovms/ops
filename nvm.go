package main

import ("fmt"
        "os"
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

// download images + mkfs from cloud storage
func downloadFile(name string) {
  // download if file not exits..  
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
  rootCmd.Execute()
}
