package checks

import (
	"bytes"
	"math/rand"
	"regexp"
	"strconv"

	"github.com/pin/tftp/v3"
)

type Tftp struct {
	checkBase
	File []TftpFile
}

type TftpFile struct {
	Name  string
	Hash  string
	Regex string
}

func (c Tftp) Run(teamID uint, boxIp string, res chan Result) {
	conn, err := tftp.NewClient(boxIp+":"+strconv.Itoa(c.Port))
	if err != nil {
		res <- Result{
			Error: "tftp connection failed",
			Debug: err.Error(),
		}
		return
	}

	if len(c.File) > 0 {
		file := c.File[rand.Intn(len(c.File))]
		wt, err := conn.Receive(file.Name, "octet")
		if err != nil {
			res <- Result{
				Error: "failed to retrieve file " + file.Name,
				Debug: err.Error(),
			}
			return
		}
		var b bytes.Buffer
		n, err := wt.WriteTo(&b)
		if err != nil {
			res <- Result{
				Error: "failed to read tftp file",
				Debug: "tried to read " + file.Name,
			}
			return
		}
		buf := b.Bytes()[:n]
		if file.Regex != "" {
			re, err := regexp.Compile(file.Regex)
			if err != nil {
				res <- Result{
					Error: "error compiling regex to match for tftp file",
					Debug: err.Error(),
				}
				return
			}
			reFind := re.Find(buf)
			if reFind == nil {
				res <- Result{
					Error: "couldn't find regex in file",
					Debug: "couldn't find regex \"" + file.Regex + "\" for " + file.Name,
				}
				return
			}
		} else if file.Hash != "" {
			fileHash, err := StringHash(string(buf))
			if err != nil {
				res <- Result{
					Error: "error calculating file hash",
					Debug: err.Error(),
				}
				return
			} else if fileHash != file.Hash {
				res <- Result{
					Error: "file hash did not match",
					Debug: "file hash " + fileHash + " did not match specified hash " + file.Hash,
				}
				return
			}
		}
	}

	res <- Result{
		Status: true,
	}
}
