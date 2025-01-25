/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dsnet/compress/bzip2"
	"github.com/spf13/cobra"
	"github.com/ulikunitz/xz"
)

var path string
var algorithm string
var archiveName string

// поддерживаемые типы архивов
const (
	zip_    string = ".zip"
	tar_    string = ".tar"
	tarGz_  string = ".tar.gz"
	tarBz2_ string = ".tar.bz2"
	tarXz_  string = ".tar.xz"
)

// алгоритмы сжатия
const (
	gzip_    string = "gzip"
	bzip2_   string = "bzip2"
	xz_      string = "xz"
	none_    string = "none"
	deflate_ string = "deflate"
	lzma_    string = "lzma"
)

// позже расширить, добавив 7z и RAR
var supportedArchiveTypes = map[string][]string{
	zip_:    {deflate_, lzma_},
	tar_:    {gzip_, bzip2_, xz_, none_}, // "none" указывает что архив не использует алгоритм сжатия
	tarGz_:  {gzip_},
	tarBz2_: {bzip2_},
	tarXz_:  {xz_},
}

var archiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "A brief description",
	Long:  `A longer description.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		createArchive(args)
	},
}

func init() {
	setupZipCommand()
	rootCmd.AddCommand(archiveCmd)
}

// базовый метод для начала этапа создания архива
func createArchive(args []string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from:", r)
			os.Exit(1)
		}
	}()
	validateInputs(args)

	archiveFile, err := os.Create(archiveName)
	if err != nil {
		panic("error create archiveFile " + err.Error())
	}
	defer archiveFile.Close()

	if strings.HasSuffix(archiveName, zip_) {
		createZIPArchive(archiveFile, args)
	} else if strings.Contains(archiveName, tar_) {
		createTARArchive(archiveFile, args)
	}
}

// базовый метод для определения слгоритма
func createTARArchive(archiveFile *os.File, args []string) {

	if algorithm == none_ {
		createTARArchiveNONE(archiveFile, args)
	} else if algorithm == gzip_ {
		createTARArchiveGZIP(archiveFile, args)
	} else if algorithm == bzip2_ {
		createTARArchiveBZIP2(archiveFile, args)
	} else if algorithm == xz_ {
		createTARArchiveXZ(archiveFile, args)
	}
}

// TAR-none
func createTARArchiveNONE(archiveFile *os.File, args []string) {
	writer := tar.NewWriter(archiveFile)
	for _, filepath := range args {
		info, err := os.Stat(filepath)
		if err != nil {
			panic("error get stat")
		}

		if info.IsDir() {
			copyTARArchiveDir(writer, filepath)
		} else {
			copyTARArchiveFile(writer, filepath, filepath)
		}
	}
}

// TAR-none копирование директории
func copyTARArchiveDir(writer *tar.Writer, pathToFile string) {
	fs.WalkDir(os.DirFS(pathToFile), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			panic("error walk to dir")
		}
		if path == "." {
			return nil
		}

		if d.IsDir() {
			header := tar.Header{
				Name:     path + "/",
				Mode:     0755,
				ModTime:  time.Now(),
				Typeflag: tar.TypeDir,
			}
			err := writer.WriteHeader(&header)
			if err != nil {
				panic("error write header")
			}
		} else {
			copyTARArchiveFile(writer, filepath.Join(pathToFile, path), path)
		}
		return nil
	})
}

// TAR-none копирование файла
func copyTARArchiveFile(writer *tar.Writer, filepath string, archiveFilePath string) {
	file, err := os.Open(filepath)
	if err != nil {
		panic("error open file for reading")
	}
	defer file.Close()

	info, err := os.Stat(filepath)
	if err != nil {
		panic("error get stat")
	}
	// используется когда аргумент - файл. требуется исключительно при вызове из функции createTARArchiveNONE(writer *tar.Writer, args []string)
	var header *tar.Header
	var fileName string
	if filepath == archiveFilePath {
		fileName = info.Name()
	} else {
		fileName = archiveFilePath
	}

	header = &tar.Header{
		Name:     fileName,
		Mode:     0644,
		Size:     info.Size(),
		ModTime:  info.ModTime(),
		Typeflag: tar.TypeReg,
	}

	err = writer.WriteHeader(header)
	if err != nil {
		panic("error write header")
	}

	_, err = io.Copy(writer, file)
	if err != nil {
		panic("error copy")
	}
}

// TAR-XZ
func createTARArchiveXZ(archiveFile *os.File, args []string) {
	xzWriter, err := xz.NewWriter(archiveFile)
	if err != nil {
		panic("error create xz writer")
	}
	defer xzWriter.Close()

	writer := tar.NewWriter(xzWriter)
	defer writer.Close()

	for _, arg := range args {
		if info, err := os.Stat(arg); err == nil {
			if info.IsDir() {
				copyTARArchiveDir(writer, arg)
			} else {
				copyTARArchiveFile(writer, arg, arg)
			}
		} else {
			panic("error get stat")
		}
	}
}

// TAR-BZIP2
func createTARArchiveBZIP2(archiveFile *os.File, args []string) {
	// добавить проверку: содержит ли имя архива расширение сжатия, если не содержит - добавить
	bz2Writer, err := bzip2.NewWriter(archiveFile, nil)
	if err != nil {
		panic("error create bzip2 writer")
	}
	defer bz2Writer.Close()

	writer := tar.NewWriter(bz2Writer)
	defer writer.Close()

	for _, arg := range args {
		if info, err := os.Stat(arg); err == nil {
			if info.IsDir() {
				copyTARArchiveDir(writer, arg)
			} else {
				copyTARArchiveFile(writer, arg, arg)
			}
		} else {
			panic("error get stat")
		}
	}

}

// TAR-GZIP
func createTARArchiveGZIP(archiveFile *os.File, args []string) {
	gzipWriter := gzip.NewWriter(archiveFile)
	defer func() {
		if err := gzipWriter.Close(); err != nil {
			fmt.Println("error closing gzip writer:", err)
		}
	}()

	writer := tar.NewWriter(gzipWriter)
	defer func() {
		if err := writer.Close(); err != nil {
			fmt.Println("error closing tar writer:", err)
		}
	}()

	for _, arg := range args {
		if info, err := os.Stat(arg); err == nil {
			if info.IsDir() {
				copyTARArchiveDir(writer, arg)
			} else {
				copyTARArchiveFile(writer, arg, arg)
			}
		} else {
			panic("error get stat")
		}
	}
}

// ZIP
func createZIPArchive(archiveFile *os.File, args []string) {

	writer := zip.NewWriter(archiveFile)
	defer writer.Close()

	if algorithm == deflate_ {
		createZIPArchiveDEFLATE(writer, args)
	} else if algorithm == lzma_ {
		createZIPArchiveLZMA(writer, args)
	}
}

// ZIP-Deflate
func createZIPArchiveDEFLATE(writer *zip.Writer, args []string) {
	for _, arg := range args {

		inf, err := os.Stat(arg)
		if err != nil {
			panic("file not exists")
		}
		if inf.IsDir() {
			copyZIPArchiveDEFLATEDir(writer, arg)
		} else {
			copyZIPArchiveDEFALTEFile(arg, arg, writer)
		}
	}
}

// ZIP-Deflate копирование директории
func copyZIPArchiveDEFLATEDir(writer *zip.Writer, argPathToFOD string) {
	err := fs.WalkDir(os.DirFS(argPathToFOD), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}
		if d.IsDir() {
			_, err := writer.Create(path + "/")
			if err != nil {
				panic("error create directory")
			}
		} else {
			copyZIPArchiveDEFALTEFile(filepath.Join(argPathToFOD, path), path, writer)
		}
		return nil
	})
	if err != nil {
		panic("error walk argument dir")
	}
}

// ZIP-Deflate копирование файла
func copyZIPArchiveDEFALTEFile(argPathToFOD string, relativePath string, writer *zip.Writer) {
	var fileWriter io.Writer
	var err error
	// используется когда аргумент - файл. требуется исключительно при вызове из функции createZIPArchiveDEFLATE(writer *zip.Writer, args []string)
	if relativePath != argPathToFOD {
		fileWriter, err = writer.Create(relativePath)
	} else {
		fileWriter, err = writer.Create(filepath.Base(argPathToFOD))
	}
	if err != nil {
		panic("error create file")
	}

	fileReader, err := os.Open(argPathToFOD)
	if err != nil {
		panic("error open file for reading")
	}
	defer fileReader.Close()

	_, err = io.Copy(fileWriter, fileReader)
	if err != nil {
		panic("error copy: " + err.Error())
	}
}

// ZIP-LZMA
func createZIPArchiveLZMA(writer *zip.Writer, args []string) {
	panic("unimplemented")
}

// выполнение проверок
func validateInputs(args []string) {
	validateArchiveName()
	validatePath()
	validateExtension()
	validateArchiveType()
	validateAlgorithm()
	validateArgsFiles(args)
}

func validateArgsFiles(files []string) {
	for _, file := range files {
		_, err := os.Stat(file)
		if err != nil {
			if os.IsNotExist(err) {
				panic("file or folder not exists")
			} else {
				panic("error " + err.Error())
			}
		}
	}
}

func validateArchiveType() {
	if strings.HasSuffix(archiveName, ".tar") {
		if algorithm == gzip_ {
			archiveName += ".gz"
		} else if algorithm == bzip2_ {
			archiveName += ".bz2"
		} else if algorithm == xz_ {
			archiveName += ".xz"
		}
	}
}

func validateArchiveName() {
	if archiveName == "" {
		panic("archive name cannot be empty")
	}
	if len(archiveName) > 255 {
		panic("archive name is too long (max 255 characters)")
	}
	if strings.ContainsAny(archiveName, `<>:"/\|?*`) {
		panic("archive name contains invalid characters")
	}
}

func validatePath() {
	fileinf, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(path, os.ModePerm); err != nil {
			panic("ошибка при создании директории: " + err.Error())
		}
		return
	}
	if err != nil {
		panic("ошибка при проверке пути: " + err.Error())
	}
	if !fileinf.IsDir() {
		panic("переменная path указывает не на директорию")
	}
}

func validateAlgorithm() {
	if algorithm == "" {
		setDefaulAlghoritm()
	}

	for ext, algs := range supportedArchiveTypes {
		if strings.HasSuffix(archiveName, ext) {
			for _, al := range algs {
				if al == algorithm {
					return
				}
			}
		}
	}
	panic("algorithm to current type cannot supported")
}

func validateExtension() {
	for ext := range supportedArchiveTypes {
		if strings.HasSuffix(archiveName, ext) {
			return
		}
	}
	panic("type not supported")
}

// установка базовых значений
func setDefaulAlghoritm() {
	if strings.HasSuffix(archiveName, zip_) {
		algorithm = deflate_
	} else if strings.HasSuffix(archiveName, tar_) {
		algorithm = none_
	} else if strings.HasSuffix(archiveName, tarGz_) {
		algorithm = gzip_
	} else if strings.HasSuffix(archiveName, tarBz2_) {
		algorithm = bzip2_
	} else if strings.HasSuffix(archiveName, tarXz_) {
		algorithm = xz_
	} else {
		panic("algorithm not supported")
	}
}

func setupZipCommand() {
	currentDirectory, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error: Unable to get current directory: %v\n", err)
		os.Exit(1)
	}

	archiveCmd.Flags().StringVarP(&path, "path", "p", currentDirectory, "Set the path where the archive will be created")
	archiveCmd.Flags().StringVarP(&archiveName, "name", "n", "", "Specify the name of the archive")
	archiveCmd.Flags().StringVarP(&algorithm, "algorithm", "a", "", "sets the file compression algorithm")

	if err := archiveCmd.MarkFlagRequired("name"); err != nil {
		fmt.Printf("Error: Unable to mark flag 'name' as required: %v\n", err)
		os.Exit(1)
	}
}
