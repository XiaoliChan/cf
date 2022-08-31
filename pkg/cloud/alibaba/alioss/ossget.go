package alioss

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/teamssix/cf/pkg/util/cmdutil"
	"github.com/teamssix/cf/pkg/util/errutil"
	"github.com/teamssix/cf/pkg/util/pubutil"
	"io"
	"os"
	"path/filepath"

	"github.com/schollz/progressbar/v3"

	log "github.com/sirupsen/logrus"
)

func getObject(bucketName string, objectKey string, outputPath string) {
	if objectKey[len(objectKey)-1:] == "/" {
		pubutil.CreateFolder(returnBucketFileName(outputPath, bucketName, objectKey))
	} else {
		log.Infof("正在下载 %s 存储桶里的 %s 对象 (Downloading %s objects from %s bucket)", bucketName, objectKey, bucketName, objectKey)
		var (
			objectSize int64
			region     string
		)
		OSSCollector := &OSSCollector{}
		Buckets, _ := OSSCollector.ListBuckets()
		for _, v := range Buckets {
			if v.Name == bucketName {
				region = v.Region
			}
		}
		fd, body, oserr, outputFile := OSSCollector.ReturnBucket(bucketName, objectKey, outputPath, region)
		_, objects := OSSCollector.ListObjects(bucketName)
		for _, obj := range objects {
			if objectKey == obj.Key {
				objectSize = obj.Size
			}
		}
		bar := returnBar(objectSize)
		io.Copy(io.MultiWriter(fd, bar), body)
		body.Close()
		defer fd.Close()
		if oserr == nil {
			log.Infof("对象已被保存到 %s (The object has been saved to %s)", outputFile, outputFile)
		}
	}
}

func DownloadAllObjects(bucketName string, outputPath string) {
	var (
		objectKey  string
		region     string
		objectList []string
	)
	OSSCollector := &OSSCollector{}
	objectList = append(objectList, "all")
	Buckets, _ := OSSCollector.ListBuckets()
	for _, v := range Buckets {
		if v.Name == bucketName {
			region = v.Region
		}
	}
	_, objects := OSSCollector.ListObjects(bucketName)
	for _, o := range objects {
		objectList = append(objectList, o.Key)
	}
	prompt := &survey.Select{
		Message: "选择一个对象 (Choose a object): ",
		Options: objectList,
	}
	survey.AskOne(prompt, &objectKey)
	if objectKey == "all" {
		bar := returnBar((int64(len(objectList) - 1)))
		for _, j := range objects {
			if j.Key[len(j.Key)-1:] == "/" {
				bar.Add(1)
				pubutil.CreateFolder(returnBucketFileName(outputPath, bucketName, j.Key))
			} else {
				bar.Add(1)
				fd, body, _, _ := OSSCollector.ReturnBucket(bucketName, j.Key, outputPath, region)
				io.Copy(fd, body)
				body.Close()
				defer fd.Close()
			}
		}
		log.Infof("对象已被保存到 %s 目录下 (The object has been saved to the %s directory)", outputPath, outputPath)
	} else {
		if objectKey[len(objectKey)-1:] == "/" {
			pubutil.CreateFolder(returnBucketFileName(outputPath, bucketName, objectKey))
		} else {
			getObject(bucketName, objectKey, outputPath)
		}
	}
}

func DownloadObjects(bucketName string, objectKey string, outputPath string, ossDownloadFlushCache bool) {
	if outputPath == "./result" {
		pubutil.CreateFolder("./result")
	}
	if bucketName == "all" {
		var (
			bucketList    []string
			bucketListAll []string
		)
		bucketListAll = append(bucketListAll, "all")
		if ossDownloadFlushCache {
			OSSCollector := &OSSCollector{}
			Buckets, _ := OSSCollector.ListBuckets()
			for _, v := range Buckets {
				_, objects := OSSCollector.ListObjects(v.Name)
				if len(objects) > 0 {
					bucketList = append(bucketList, v.Name)
					bucketListAll = append(bucketListAll, v.Name)
				}
			}
		} else {
			Buckets := cmdutil.ReadCacheFile(OSSCacheFilePath, "alibaba", "OSS")
			for _, v := range Buckets {
				OSSCollector := &OSSCollector{}
				_, objects := OSSCollector.ListObjects(v[1])
				if len(objects) > 0 {
					bucketList = append(bucketList, v[1])
					bucketListAll = append(bucketListAll, v[1])
				}
			}
		}
		bucketListAll = append(bucketListAll, "exit")
		if len(bucketList) == 1 {
			bucketName = bucketList[0]
		} else {
			prompt := &survey.Select{
				Message: "选择一个存储桶 (Choose a bucket): ",
				Options: bucketListAll,
			}
			survey.AskOne(prompt, &bucketName)
		}

		if bucketName == "all" {
			for _, v := range bucketList {
				if objectKey == "all" {
					DownloadAllObjects(v, outputPath)
				} else {
					getObject(v, objectKey, outputPath)
				}
			}
		} else if bucketName == "exit" {
			os.Exit(0)
		} else {
			if objectKey == "all" {
				DownloadAllObjects(bucketName, outputPath)
			} else {
				getObject(bucketName, objectKey, outputPath)
			}
		}
	} else {
		OSSCollector := &OSSCollector{}
		_, objects := OSSCollector.ListObjects(bucketName)
		if len(objects) > 0 {
			if objectKey == "all" {
				DownloadAllObjects(bucketName, outputPath)
			} else {
				getObject(bucketName, objectKey, outputPath)
			}
		} else {
			log.Warnf("在 %s 存储桶中没有发现对象 (No object found in %s storage bucket)", bucketName, bucketName)
		}
	}
}

func (o *OSSCollector) ReturnBucket(bucketName string, objectKey string, outputPath string, region string) (*os.File, io.ReadCloser, error, string) {
	o.OSSClient(region)
	bucket, err := o.Client.Bucket(bucketName)
	errutil.HandleErr(err)
	outputFile := returnBucketFileName(outputPath, bucketName, objectKey)
	fd, oserr := os.OpenFile(outputFile, os.O_WRONLY|os.O_CREATE, 0660)
	errutil.HandleErr(oserr)
	body, err := bucket.GetObject(objectKey)
	errutil.HandleErr(err)
	return fd, body, oserr, outputFile
}

func returnBar(replen int64) *progressbar.ProgressBar {
	bar := progressbar.NewOptions64(replen,
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(false),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetDescription("Downloading..."),
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	return bar
}

func returnBucketFileName(outputPath string, bucketName string, objectName string) string {
	outputBucketFile := filepath.Join(outputPath, bucketName)
	pubutil.CreateFolder(outputBucketFile)
	outputFileName := filepath.Join(outputBucketFile, objectName)
	return outputFileName
}
