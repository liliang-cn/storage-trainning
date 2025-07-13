# Markdown to PDF Converter (md2pdf)

This is a simple, self-contained command-line tool written in Go to convert Markdown files into PDF documents.

It supports converting a single Markdown file or recursively converting all Markdown files within a directory.

## Prerequisites

None! This tool is written in pure Go and has **no external dependencies**.

## How to Use

1.  **Clone or download the project.**

2.  **Navigate to the project directory:**
    ```bash
    cd md2pdf
    ```

3.  **Install Go dependencies:**
    The tool uses `goldmark` for Markdown parsing and `gofpdf` for PDF generation. The first time you run it, Go will automatically download the necessary packages. You can also fetch them manually:
    ```bash
    go mod tidy
    ```

4.  **Run the conversion:**

    *   **To convert a single file:**
        ```bash
        go run . /path/to/your/file.md
        ```
        The output will be `file.pdf` in the same directory.

    *   **To convert all Markdown files in a directory:**
        ```bash
        go run . /path/to/your/directory
        ```
        The tool will find all `.md` files in the directory and its subdirectories and create a corresponding `.pdf` file for each.

## Example

Assuming you are in the `storage-trainning` directory:

```bash
# Convert a single file
go run ./md2pdf ./week0/day01.md

# Convert all files in the week0 directory
go run ./md2pdf ./week0
```

## How It Works

1.  The program takes a file or directory path as a command-line argument.
2.  It reads the Markdown file(s).
3.  It uses the `goldmark` library to parse the Markdown content into an Abstract Syntax Tree (AST).
4.  It walks through the AST and uses the `gofpdf` library to programmatically build a PDF document based on the Markdown structure (headings, paragraphs, lists, etc.).
5.  The resulting PDF is saved with the same name as the original Markdown file, but with a `.pdf` extension.