-- 初始化 PostgreSQL 扩展
-- 此脚本在数据库首次创建时自动执行

-- 启用 pgvector 扩展 (向量搜索)
CREATE EXTENSION IF NOT EXISTS vector;

-- 启用 PostGIS 扩展 (地理空间)
CREATE EXTENSION IF NOT EXISTS postgis;

-- 启用 PostGIS 拓扑扩展
CREATE EXTENSION IF NOT EXISTS postgis_topology;

-- 验证扩展安装
DO $$
BEGIN
    RAISE NOTICE 'Installed extensions:';
    RAISE NOTICE '  - vector: %', (SELECT extversion FROM pg_extension WHERE extname = 'vector');
    RAISE NOTICE '  - postgis: %', (SELECT extversion FROM pg_extension WHERE extname = 'postgis');
END $$;
