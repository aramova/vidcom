package main

import (
	"fmt"
	"io"
	"bufio"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// Color codes for console output
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
)

func main() {
	// Set up signal handling for graceful exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n" + colorYellow + "WARNING:" + colorReset + " Ctrl+C detected. Exiting gracefully...")
		os.Exit(1) // Exit the program
	}()

	// Open log file for writing
	logFile, err := os.OpenFile("compression_report.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	defer logFile.Close()

	// Create a logger that writes only to the log file for FFmpeg output
	ffmpegLogger := log.New(logFile, "", log.LstdFlags)

	// Create a logger that writes to both console and log file for status messages
	statusLogger := log.New(io.MultiWriter(os.Stdout, logFile), "", log.LstdFlags)

	// Find all video files and count subdirectories
	videoFiles, dirCount := findVideoFiles(".")
	statusLogger.Printf(colorGreen+"INFO:"+colorReset+" Found %d video files to process in %d subdirectories.\n", len(videoFiles), dirCount)

	// Process each video file
	totalOriginalSize, totalCompressedSize := processVideos(videoFiles, ffmpegLogger, statusLogger)

	// Calculate and display total space saved
	totalSpaceSaved := totalOriginalSize - totalCompressedSize
	percentageSaved := (float64(totalSpaceSaved) / float64(totalOriginalSize)) * 100

	statusLogger.Println("------------------------------------")
	statusLogger.Printf("Total Space Saved: %.2f MB (%.2f%%)\n", float64(totalSpaceSaved)/1024/1024, percentageSaved)
	statusLogger.Println("------------------------------------")
}

// findVideoFiles recursively searches a directory for video files and counts subdirectories.
func findVideoFiles(dirPath string) ([]string, int) {
	var files []string
	dirCount := 0
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			dirCount++
		} else if isVideoFile(path) {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		fmt.Println("Error:", err)
	}
	return files, dirCount - 1 // Subtract 1 to exclude the starting directory itself
}

// processVideos compresses a list of video files.
func processVideos(videoFiles []string, ffmpegLogger, statusLogger *log.Logger) (int64, int64) {
	var totalOriginalSize, totalCompressedSize int64
	for _, filePath := range videoFiles {
		originalSize, compressedSize := processVideo(filePath, ffmpegLogger, statusLogger)
		totalOriginalSize += originalSize
		totalCompressedSize += compressedSize
	}
	return totalOriginalSize, totalCompressedSize
}

// isVideoFile checks if a file is a video file based on its extension.
func isVideoFile(filePath string) bool {
	fileExtension := strings.ToLower(filepath.Ext(filePath))
	return fileExtension == ".mp4" || fileExtension == ".mov"
}

// isCompressedFile checks if a file is already compressed based on its path.
func isCompressedFile(filePath string) bool {
	return strings.Contains(strings.ToLower(filePath), "completed")
}

// processVideo compresses a single video file and handles compression results.
func processVideo(filePath string, ffmpegLogger, statusLogger *log.Logger) (int64, int64) {
	statusLogger.Println(colorGreen + "INFO:" + colorReset + " Starting work on", filePath)

	// Get file information
	fileDir, fileName := filepath.Split(filePath)

	// Create the "Completed" directory if it doesn't exist
	completedDir := filepath.Join(fileDir, "Completed")
	if _, err := os.Stat(completedDir); os.IsNotExist(err) {
		os.Mkdir(completedDir, 0755)
	}

	// Construct the compressed file path (inside "Completed")
	compressedFilePath := filepath.Join(completedDir, fileName)

	// Check if the compressed file already exists
	if _, err := os.Stat(compressedFilePath); err == nil {
		statusLogger.Printf(colorYellow+"WARNING:"+colorReset+" %s already exists in Completed directory. Skipping.\n", fileName)
		return 0, 0 // Skip if compressed file exists
	}

	// Get the original file size
	originalFileInfo, err := os.Stat(filePath)
	if err != nil {
		statusLogger.Println(colorYellow+"WARNING:"+colorReset+" Error getting original file size:", err)
		return 0, 0
	}
	originalFileSize := originalFileInfo.Size()

	// Compress the video using FFmpeg with hevc_videotoolbox
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c:v", "hevc_videotoolbox", "-c:a", "copy", compressedFilePath)

	// Capture FFmpeg output to a pipe
	cmdOutput, err := cmd.StderrPipe()
	if err != nil {
		statusLogger.Println(colorYellow+"WARNING:"+colorReset+" Error creating output pipe:", err)
		return originalFileSize, 0
	}

	// Start the FFmpeg command
	if err := cmd.Start(); err != nil {
		statusLogger.Println(colorYellow+"WARNING:"+colorReset+" Error starting FFmpeg:", err)
		return originalFileSize, 0
	}

	// Read and log FFmpeg output (to log file only)
	scanner := bufio.NewScanner(cmdOutput)
	// Increase the scanner's buffer size (e.g., to 64KB)
	const maxCapacity = 256 * 1024 // Your required line length
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	// Continue with reading the output
	for scanner.Scan() {
		ffmpegLogger.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		statusLogger.Println(colorYellow+"WARNING:"+colorReset+" Error reading FFmpeg output:", err)
	}

	// Wait for FFmpeg to finish
	if err := cmd.Wait(); err != nil {
		statusLogger.Println(colorYellow+"WARNING:"+colorReset+" Error waiting for FFmpeg to finish:", err)
		return originalFileSize, 0
	}

	// Get compressed file size
	compressedFileInfo, err := os.Stat(compressedFilePath)
	if err != nil {
		statusLogger.Println(colorYellow+"WARNING:"+colorReset+" Error getting compressed file size:", err)
		return originalFileSize, 0
	}
	compressedFileSize := compressedFileInfo.Size()

	// Handle compression results
	if compressedFileSize >= originalFileSize {
		// Compressed file is larger or the same size, so delete it
		os.Remove(compressedFilePath)
		if compressedFileSize > originalFileSize {
			statusLogger.Printf(colorYellow+"WARNING:"+colorReset+" %s is larger than original %s by %.2f MB (%.2f%% increase). Deleting compressed file.\n",
				compressedFilePath, filePath,
				float64(compressedFileSize-originalFileSize)/1024/1024,
				(float64(compressedFileSize-originalFileSize)/float64(originalFileSize))*100)
		} else {
			statusLogger.Printf(colorYellow+"WARNING:"+colorReset+" %s is the same size as the original %s. Deleting compressed file.\n",
				compressedFilePath, filePath)
		}
		return originalFileSize, originalFileSize // Return original size since we're keeping the original
	}

	// Compressed file is smaller, log the success and return sizes
	statusLogger.Printf(colorBlue+"Compressed:"+colorReset+" '%s' to '%s' - Original size: %.2f MB, Compressed size: %.2f MB (%.2f%% reduction)\n",
		filePath, compressedFilePath,
		float64(originalFileSize)/1024/1024, float64(compressedFileSize)/1024/1024,
		(float64(originalFileSize-compressedFileSize)/float64(originalFileSize))*100)

	return originalFileSize, compressedFileSize
}

// handleError handles errors gracefully.
func handleError(err error) {
	if err != nil {
		fmt.Println("Error:", err)
	}
}

// formatFileSize converts bytes to a human-readable file size string.
func formatFileSize(fileSize int64) string {
	if fileSize < 1024 {
		return fmt.Sprintf("%d B", fileSize)
	} else if fileSize < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(fileSize)/1024)
	} else if fileSize < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(fileSize)/1024/1024)
	} else {
		return fmt.Sprintf("%.2f GB", float64(fileSize)/1024/1024/1024)
	}
}

// stringToInt64 converts a string to an int64, handling any errors.
func stringToInt64(str string) int64 {
	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0 // Return 0 on error
	}
	return i
}