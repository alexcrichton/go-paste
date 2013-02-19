package image

import "io"
import "os"
import "os/exec"

func hascmd(cmd string, args ...string) bool {
  return exec.Command(cmd, args...).Run() == nil
}

func cp(src, dst string) error {
  out, err := os.Create(dst)
  if err != nil { return err }
  defer out.Close()

  in, err := os.Open(src)
  if err != nil { return err }
  defer in.Close()

  io.Copy(out, in)
  return nil
}
