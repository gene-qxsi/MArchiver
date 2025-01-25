/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"archive/tar"
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
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
	gzip    string = "gzip"
	bzip2   string = "bzip2"
	xz      string = "xz"
	none    string = "none"
	deflate string = "deflate"
	lzma    string = "lzma"
)

// позже расширить, добавив 7z и RAR
var supportedArchiveTypes = map[string][]string{
	zip_:    {deflate, lzma},
	tar_:    {gzip, bzip2, xz, none}, // "none" указывает что архив не использует алгоритм сжатия
	tarGz_:  {gzip},
	tarBz2_: {bzip2},
	tarXz_:  {xz},
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

	writer := tar.NewWriter(archiveFile)

	if algorithm == none {
		createTARArchiveNONE(writer, args)
	} else if algorithm == gzip {
		createTARArchiveGZIP(writer, args)
	} else if algorithm == bzip2 {
		createTARArchiveBZIP2(writer, args)
	} else if algorithm == xz {
		createTARArchiveXZ(writer, args)
	}
}

// TAR-none
func createTARArchiveNONE(writer *tar.Writer, args []string) {
	for _, filepath := range args {
		info, err := os.Stat(filepath)
		if err != nil {
			panic("error get stat")
		}

		if info.IsDir() {
			copyTARArchiveNONEDir(writer, filepath)
		} else {
			copyTARArchiveNONEFile(writer, filepath, filepath)
		}
	}
}

// TAR-none копирование директории
func copyTARArchiveNONEDir(writer *tar.Writer, pathToFile string) {
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
			copyTARArchiveNONEFile(writer, filepath.Join(pathToFile, path), path)
		}
		return nil
	})
}

// TAR-none копирование файла
func copyTARArchiveNONEFile(writer *tar.Writer, filepath string, archiveFilePath string) {
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

func createTARArchiveXZ(writer *tar.Writer, args []string) {
	// реализовать 24.01
}

func createTARArchiveBZIP2(writer *tar.Writer, args []string) {
	// реализовать 24.01
}

func createTARArchiveGZIP(writer *tar.Writer, args []string) {
	// реализовать 24.01
}

// ZIP
func createZIPArchive(archiveFile *os.File, args []string) {

	writer := zip.NewWriter(archiveFile)
	defer writer.Close()

	if algorithm == deflate {
		createZIPArchiveDEFLATE(writer, args)
	} else if algorithm == lzma {
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
	fmt.Println(argPathToFOD)
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
		algorithm = deflate
	} else if strings.HasSuffix(archiveName, tar_) {
		algorithm = none
	} else if strings.HasSuffix(archiveName, tarGz_) {
		algorithm = gzip
	} else if strings.HasSuffix(archiveName, tarBz2_) {
		algorithm = bzip2
	} else if strings.HasSuffix(archiveName, tarXz_) {
		algorithm = xz
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
