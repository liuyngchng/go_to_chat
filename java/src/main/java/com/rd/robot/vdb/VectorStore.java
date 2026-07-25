package com.rd.robot.vdb;

import com.rd.robot.model.SearchResult;
import com.rd.robot.model.VectorRecord;

import java.util.List;

public interface VectorStore {

    void ensureCollection(int dimension) throws Exception;

    void insert(List<VectorRecord> records) throws Exception;

    List<SearchResult> search(double[] queryVector, int topK, double scoreThreshold) throws Exception;

    void deleteByIds(List<String> ids) throws Exception;

    void deleteBySource(String source) throws Exception;

    void close() throws Exception;

    void purge() throws Exception;
}
