package com.rd.robot.knowledge;

import org.apache.pdfbox.Loader;
import org.apache.pdfbox.pdmodel.PDDocument;
import org.apache.pdfbox.text.PDFTextStripper;
import org.apache.poi.ss.usermodel.*;
import org.apache.poi.xssf.usermodel.XSSFWorkbook;
import org.apache.poi.xwpf.usermodel.XWPFDocument;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.*;
import java.nio.file.Files;
import java.nio.file.Path;

/**
 * Text extractor for various document formats (PDF, DOCX, XLSX).
 */
public class TextExtractor {

    private static final Logger log = LoggerFactory.getLogger(TextExtractor.class);

    /**
     * Extract text from file, with caching to .txt sidecar file.
     */
    public static String extractAndSaveText(String filePath) throws Exception {
        String txtPath = filePath + ".txt";
        File txtFile = new File(txtPath);
        if (txtFile.exists()) {
            String cached = Files.readString(Path.of(txtPath));
            if (cached != null && !cached.trim().isEmpty()) {
                return cached;
            }
        }

        String text = extractText(filePath);

        try {
            Files.writeString(Path.of(txtPath), text);
        } catch (IOException ignored) {}

        return text;
    }

    private static String extractText(String filePath) throws Exception {
        String ext = filePath.toLowerCase();
        if (ext.endsWith(".pdf")) return extractPDF(filePath);
        else if (ext.endsWith(".docx")) return extractDOCX(filePath);
        else if (ext.endsWith(".xlsx") || ext.endsWith(".xls")) return extractXLSX(filePath);
        throw new IllegalArgumentException("不支持的文件格式: " + ext);
    }

    private static String extractPDF(String filePath) throws Exception {
        try (PDDocument document = Loader.loadPDF(new File(filePath))) {
            int numPages = document.getNumberOfPages();
            if (numPages == 0) throw new RuntimeException("PDF 文件为空");

            PDFTextStripper stripper = new PDFTextStripper();
            String text = stripper.getText(document);
            String result = text != null ? text.trim() : "";
            if (result.isEmpty()) throw new RuntimeException("PDF 文件无文本内容，可能是扫描件");
            return result;
        }
    }

    private static String extractDOCX(String filePath) throws Exception {
        try (FileInputStream fis = new FileInputStream(filePath);
             XWPFDocument document = new XWPFDocument(fis)) {

            StringBuilder sb = new StringBuilder();
            for (var paragraph : document.getParagraphs()) {
                String text = paragraph.getText();
                if (text != null && !text.trim().isEmpty()) {
                    sb.append(text.trim()).append("\n");
                }
            }
            String result = sb.toString().trim();
            if (result.isEmpty()) throw new RuntimeException("DOCX 文件无文本内容");
            return result;
        }
    }

    private static String extractXLSX(String filePath) throws Exception {
        if (filePath.toLowerCase().endsWith(".xls")) {
            throw new RuntimeException("旧版 .xls 格式不支持，请转换为 .xlsx 格式后再上传");
        }

        try (FileInputStream fis = new FileInputStream(filePath);
             XSSFWorkbook workbook = new XSSFWorkbook(fis)) {

            StringBuilder sb = new StringBuilder();
            for (int si = 0; si < workbook.getNumberOfSheets(); si++) {
                Sheet sheet = workbook.getSheetAt(si);
                for (Row row : sheet) {
                    StringBuilder rowStr = new StringBuilder();
                    boolean hasContent = false;
                    for (Cell cell : row) {
                        String val = getCellValue(cell);
                        if (val != null && !val.isEmpty()) {
                            if (!rowStr.isEmpty()) rowStr.append("\t");
                            rowStr.append(val);
                            hasContent = true;
                        }
                    }
                    if (hasContent) sb.append(rowStr).append("\n");
                }
            }
            String result = sb.toString().trim();
            if (result.isEmpty()) throw new RuntimeException("XLSX 文件无内容");
            return result;
        }
    }

    private static String getCellValue(Cell cell) {
        if (cell == null) return "";
        return switch (cell.getCellType()) {
            case STRING -> cell.getStringCellValue();
            case NUMERIC -> {
                if (DateUtil.isCellDateFormatted(cell)) {
                    yield cell.getLocalDateTimeCellValue().toString();
                }
                double val = cell.getNumericCellValue();
                if (val == Math.floor(val) && !Double.isInfinite(val)) {
                    yield String.valueOf((long) val);
                }
                yield String.valueOf(val);
            }
            case BOOLEAN -> String.valueOf(cell.getBooleanCellValue());
            case FORMULA -> {
                try { yield cell.getStringCellValue(); }
                catch (Exception e) { yield String.valueOf(cell.getNumericCellValue()); }
            }
            default -> "";
        };
    }
}