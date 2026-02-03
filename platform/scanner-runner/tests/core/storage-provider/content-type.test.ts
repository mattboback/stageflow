import { describe, expect, it } from "vitest";

import { guessContentType } from "../../../src/core/storage-provider/content-type";

describe("guessContentType", () => {
  describe("JSON files", () => {
    it("returns application/json for .json files", () => {
      expect(guessContentType("results.json")).toBe("application/json");
      expect(guessContentType("/path/to/data.json")).toBe("application/json");
    });
  });

  describe("HTML files", () => {
    it("returns text/html for .html files", () => {
      expect(guessContentType("report.html")).toBe("text/html");
      expect(guessContentType("/reports/index.html")).toBe("text/html");
    });

    it("returns text/html for .htm files", () => {
      expect(guessContentType("page.htm")).toBe("text/html");
    });
  });

  describe("CSS files", () => {
    it("returns text/css for .css files", () => {
      expect(guessContentType("styles.css")).toBe("text/css");
    });
  });

  describe("JavaScript files", () => {
    it("returns application/javascript for .js files", () => {
      expect(guessContentType("script.js")).toBe("application/javascript");
    });
  });

  describe("image files", () => {
    it("returns image/png for .png files", () => {
      expect(guessContentType("screenshot.png")).toBe("image/png");
    });

    it("returns image/jpeg for .jpg files", () => {
      expect(guessContentType("photo.jpg")).toBe("image/jpeg");
    });

    it("returns image/jpeg for .jpeg files", () => {
      expect(guessContentType("photo.jpeg")).toBe("image/jpeg");
    });

    it("returns image/gif for .gif files", () => {
      expect(guessContentType("animation.gif")).toBe("image/gif");
    });

    it("returns image/webp for .webp files", () => {
      expect(guessContentType("modern.webp")).toBe("image/webp");
    });

    it("returns image/svg+xml for .svg files", () => {
      expect(guessContentType("icon.svg")).toBe("image/svg+xml");
    });
  });

  describe("other files", () => {
    it("returns application/zip for .zip files", () => {
      expect(guessContentType("archive.zip")).toBe("application/zip");
    });

    it("returns text/plain for .txt files", () => {
      expect(guessContentType("readme.txt")).toBe("text/plain");
    });

    it("returns application/xml for .xml files", () => {
      expect(guessContentType("config.xml")).toBe("application/xml");
    });
  });

  describe("case insensitivity", () => {
    it("handles uppercase extensions", () => {
      expect(guessContentType("file.JSON")).toBe("application/json");
      expect(guessContentType("file.PNG")).toBe("image/png");
      expect(guessContentType("file.HTML")).toBe("text/html");
    });

    it("handles mixed case extensions", () => {
      expect(guessContentType("file.Json")).toBe("application/json");
      expect(guessContentType("file.Png")).toBe("image/png");
    });
  });

  describe("fallback behavior", () => {
    it("returns application/octet-stream for unknown extensions", () => {
      expect(guessContentType("file.xyz")).toBe("application/octet-stream");
      expect(guessContentType("file.unknown")).toBe("application/octet-stream");
    });

    it("returns application/octet-stream for files without extension", () => {
      expect(guessContentType("Makefile")).toBe("application/octet-stream");
      expect(guessContentType("README")).toBe("application/octet-stream");
    });

    it("returns application/octet-stream for empty string", () => {
      expect(guessContentType("")).toBe("application/octet-stream");
    });
  });

  describe("path handling", () => {
    it("extracts extension from full paths", () => {
      expect(guessContentType("/home/user/documents/report.pdf")).toBe(
        "application/octet-stream",
      );
      expect(guessContentType("/var/www/html/index.html")).toBe("text/html");
    });

    it("handles paths with multiple dots", () => {
      expect(guessContentType("file.backup.json")).toBe("application/json");
      expect(guessContentType("image.thumb.png")).toBe("image/png");
    });

    it("handles hidden files with extensions", () => {
      expect(guessContentType(".config.json")).toBe("application/json");
    });
  });
});
