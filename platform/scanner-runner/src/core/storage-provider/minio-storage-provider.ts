/**
 * MinIO-based storage provider for uploading scan artifacts.
 */

import fg from "fast-glob";
import { Client as MinioClient } from "minio";
import { join } from "node:path";

import type { ScannerLogger, StorageConfig, StorageProvider } from "../types";

import { createLogger } from "../../utils/logger";
import { withRetry, withTimeout } from "./async";
import { guessContentType } from "./content-type";
import { parseMinioEndpoint } from "./endpoint";
import { getFileSize, waitForFileReady } from "./files";
import { wrapMinioError } from "./minio-errors";

export class MinioStorageProvider implements StorageProvider {
  private readonly client: MinioClient;
  private readonly logger: ScannerLogger;
  private readonly connection: { endpoint: string; port: number; useSSL: boolean };

  constructor(config: StorageConfig, logger?: ScannerLogger) {
    const { endPoint, port } = parseMinioEndpoint(config.endpoint, config.useSSL);

    this.client = new MinioClient({
      endPoint,
      port,
      useSSL: config.useSSL,
      accessKey: config.accessKey,
      secretKey: config.secretKey,
    });

    this.logger = logger ?? createLogger("StorageProvider");
    this.connection = { endpoint: endPoint, port, useSSL: config.useSSL };
  }

  async ensureBucket(bucket: string): Promise<void> {
    this.logger.info("Ensuring MinIO bucket", {
      bucket,
      endpoint: this.connection.endpoint,
      port: this.connection.port,
      useSSL: this.connection.useSSL,
    });

    const exists = await withRetry({
      action: "bucketExists",
      fn: () => this.client.bucketExists(bucket),
      logger: this.logger,
      connection: this.connection,
      context: { bucket },
    });
    if (exists) {
      return;
    }

    await withRetry({
      action: "makeBucket",
      fn: () => this.client.makeBucket(bucket, ""),
      logger: this.logger,
      connection: this.connection,
      context: { bucket },
    });
    this.logger.info("Created bucket", { bucket });
  }

  async upload(
    bucket: string,
    key: string,
    filePath: string,
    contentType?: string,
  ): Promise<void> {
    const ct = contentType ?? guessContentType(filePath);
    const sizeBytes = await getFileSize(filePath);
    const startedAt = Date.now();

    this.logger.debug("Uploading file", {
      bucket,
      key,
      contentType: ct,
      sizeBytes,
    });

    const timeoutMs = 30_000;
    try {
      const uploadPromise = this.client.fPutObject(bucket, key, filePath, {
        "Content-Type": ct,
      });

      await withTimeout(uploadPromise, timeoutMs, `Upload timed out for ${key}`);
    } catch (err) {
      throw wrapMinioError(this.connection, "upload", err, { bucket, key, filePath });
    }

    this.logger.debug("Uploaded file", {
      bucket,
      key,
      contentType: ct,
      sizeBytes,
      durationMs: Date.now() - startedAt,
    });
  }

  async uploadBuffer(
    bucket: string,
    key: string,
    data: Buffer,
    contentType?: string,
  ): Promise<void> {
    const timeoutMs = 30_000;
    try {
      const uploadPromise = this.client.putObject(bucket, key, data, data.length, {
        "Content-Type": contentType ?? "application/octet-stream",
      });

      await withTimeout(uploadPromise, timeoutMs, `Buffer upload timed out for ${key}`);
      this.logger.debug("Uploaded buffer", { bucket, key, size: data.length });
    } catch (err) {
      throw wrapMinioError(this.connection, "uploadBuffer", err, { bucket, key });
    }
  }

  async uploadDirectory(
    bucket: string,
    prefix: string,
    localDir: string,
  ): Promise<number> {
    const files = await fg(["**/*"], {
      cwd: localDir,
      dot: true,
      onlyFiles: true,
    });

    if (files.length === 0) {
      return 0;
    }

    this.logger.info("Uploading directory", {
      localDir,
      prefix,
      fileCount: files.length,
    });

    let uploadedCount = 0;
    let skippedCount = 0;

    for (const relPath of files) {
      const normalized = relPath.split("\\").join("/");
      const objectName = `${prefix}/${normalized}`;
      const filePath = join(localDir, relPath);

      const isReady = await waitForFileReady(filePath, this.logger);
      if (!isReady) {
        this.logger.warn("Skipping file after retries - still empty or unreadable", {
          filePath,
        });
        skippedCount += 1;
        continue;
      }

      try {
        await this.upload(bucket, objectName, filePath);
        uploadedCount += 1;
      } catch (error) {
        this.logger.error("Failed to upload file", {
          filePath,
          error: error instanceof Error ? error.message : String(error),
        });
        skippedCount += 1;
      }
    }

    if (skippedCount > 0) {
      this.logger.warn("Upload summary", {
        total: files.length,
        uploaded: uploadedCount,
        skipped: skippedCount,
      });
    }

    return uploadedCount;
  }

  async download(bucket: string, key: string, destPath: string): Promise<void> {
    try {
      await this.client.fGetObject(bucket, key, destPath);
      this.logger.debug("Downloaded file", { bucket, key, destPath });
    } catch (err) {
      throw wrapMinioError(this.connection, "download", err, { bucket, key, destPath });
    }
  }

  async exists(bucket: string, key: string): Promise<boolean> {
    try {
      await this.client.statObject(bucket, key);
      return true;
    } catch {
      return false;
    }
  }

  getClient(): MinioClient {
    return this.client;
  }
}
