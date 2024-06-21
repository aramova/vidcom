```
# Go Video Compressor

This is a Go program that recursively compresses video files (.mp4 and .mov) in a directory and its subdirectories using FFmpeg and the `hevc_videotoolbox` codec (hardware-accelerated on macOS). It includes features like progress display, logging, error handling, and file size comparison.

## Features

- **Recursive Compression:** Traverses all directories and subdirectories.
- **File Type Filtering:** Compresses only .mp4 and .mov files (case-insensitive).
- **Skipping Compressed Files:** Skips files in directories named "Completed" or if a compressed version already exists.
- **FFmpeg with `hevc_videotoolbox`:** Uses hardware-accelerated encoding on macOS for faster compression.
- **Custom Progress Bar:** Displays a dynamic progress bar in the console during compression.
- **Detailed Logging:**  Logs all actions (information, warnings, errors, FFmpeg output) to a `compression_report.txt` file.
- **Size Comparison:** Compares the original and compressed file sizes, displaying the difference in MB and as a percentage.
- **Larger Compressed File Handling:** Deletes compressed files that are larger than the original and logs a warning.
- **Graceful Exit (Ctrl+C):** Handles Ctrl+C to terminate the script and any running FFmpeg processes.
- **Colored Output:** Uses colors in the console output to distinguish different types of messages.
- **Total Space Saved Calculation:** Calculates and displays the total space saved after compression.

## Requirements

- **Go:**  Install Go from [https://golang.org/](https://golang.org/)
- **FFmpeg:** Install FFmpeg (with `hevc_videotoolbox` support). You can download it from [https://ffmpeg.org/](https://ffmpeg.org/). On macOS, you can install it using Homebrew:
  ```bash
  brew install ffmpeg
  ```

## How to Run

1. **Save the Code:** Save the Go code as `vidcom.go`.

2. **Initialize Go Module:**
   ```bash
   go mod init vidcom 
   ```

3. **Build the Executable:**
   ```bash
   go build
   ```

4. **Run the Program:**
   ```bash
   ./vidcom
   ```

   The script will start processing video files in the current directory and its subdirectories.

## Logic Flow

1. **Initialization:**
   - Set up signal handling for Ctrl+C (SIGINT).
   - Open the `compression_report.txt` log file.
   - Create loggers for FFmpeg output (log file only) and status messages (console and log file).

2. **File Discovery:**
   - Use `findVideoFiles` to recursively search the directory for .mp4 and .mov files and count subdirectories.
   - Log the number of video files and subdirectories found.

3. **Video Processing:**
   - Iterate through the list of video files found.
   - For each video file, call the `processVideo` function.

4. **`processVideo` Function:**
   - Log the start of processing for the current file.
   - Create the "Completed" subdirectory if it doesn't exist.
   - Construct the path for the compressed file (inside the "Completed" directory).
   - Check if the compressed file already exists and skip compression if it does.
   - Get the original file size.
   - Execute the FFmpeg command using `hevc_videotoolbox` for compression.
   - Capture FFmpeg's output (standard error) to a pipe.
   - Start the FFmpeg command in a separate goroutine.
   - **Progress Bar Goroutine:**
     - Read FFmpeg's output line by line.
     - Log the output to the log file.
     - Parse relevant lines (containing `size=`) to extract progress information.
     - Update the console progress bar using the `displayProgressBar` function.
   - Wait for FFmpeg to complete.
   - Get the compressed file size.
   - Handle compression results:
     - If the compressed file is smaller, log the success and file size reduction.
     - If the compressed file is larger or the same size, delete it and log a warning.

5. **Final Report:**
   - After processing all files, calculate and display the total space saved in MB and as a percentage.

## Technical Details

- **FFmpeg Command:** 
  The script uses the following FFmpeg command for compression:
  ```bash
  ffmpeg -i [input_file] -c:v hevc_videotoolbox -c:a copy -progress pipe:2 [output_file] 
  ```
  - `-i [input_file]`:  Specifies the input video file.
  - `-c:v hevc_videotoolbox`: Uses the hardware-accelerated `hevc_videotoolbox` codec for video encoding.
  - `-c:a copy`: Copies the audio stream without re-encoding.
  - `-progress pipe:2`: Sends progress information to the standard error stream (stderr), which is captured by the Go program.
  - `[output_file]`: Specifies the path for the compressed output file.

- **Progress Bar:** 
  The `displayProgressBar` function creates a simple text-based progress bar in the console, updating in real-time based on the size information extracted from FFmpeg's output.

- **Logging:**
  The script uses two loggers:
  - `ffmpegLogger`: Logs detailed FFmpeg output to the `compression_report.txt` file.
  - `statusLogger`: Logs status updates, warnings, and errors to both the console and the `compression_report.txt` file.

- **Error Handling:**
  The script includes error handling throughout the code to catch and log errors that may occur during file processing, FFmpeg execution, or file size operations.

## Notes

- The progress bar implementation is basic and assumes that file size is a reasonable indicator of progress. You may need to adjust the parsing logic to extract more accurate progress data from FFmpeg's output if needed.
- The compression settings used in the FFmpeg command (e.g., the `hevc_videotoolbox` codec) are suitable for macOS. You may need to modify them for other operating systems.
- The script currently processes all video files found. Consider adding options for filtering files based on other criteria (e.g., file size, modification date) if required.

## License

This project is licensed under the Apache License Version 2.0. See the `LICENSE` file for details.

```
