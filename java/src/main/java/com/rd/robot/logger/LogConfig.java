package com.rd.robot.logger;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Custom log configuration.
 * Enhances the default log4j2 pattern with source file info.
 *
 * In production, configure log4j2.xml to use:
 * %d{yyyy-MM-dd HH:mm:ss.SSS} [%level] [%c{1}:%L] %msg%n
 */
public class LogConfig {

    private static final Logger log = LoggerFactory.getLogger(LogConfig.class);

    private LogConfig() {}

    /**
     * Log an info message with source location.
     */
    public static void info(String message, Object... args) {
        log.info(message, args);
    }

    /**
     * Log a warning message with source location.
     */
    public static void warn(String message, Object... args) {
        log.warn(message, args);
    }

    /**
     * Log an error message with source location.
     */
    public static void error(String message, Object... args) {
        log.error(message, args);
    }

    /**
     * Log a debug message with source location.
     */
    public static void debug(String message, Object... args) {
        log.debug(message, args);
    }
}