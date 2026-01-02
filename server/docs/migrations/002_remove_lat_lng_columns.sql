-- 移除废弃的 latitude 和 longitude 列
-- 执行此脚本前请确保 001_add_postgis_location.sql 已成功执行
-- 所有经纬度数据应已迁移到 location 列

-- 删除旧的 latitude 和 longitude 列
ALTER TABLE images DROP COLUMN IF EXISTS latitude;
ALTER TABLE images DROP COLUMN IF EXISTS longitude;

-- 删除可能存在的旧触发器（如果之前有基于 latitude/longitude 的触发器）
DROP TRIGGER IF EXISTS trigger_update_image_location ON images;
DROP FUNCTION IF EXISTS update_image_location();

-- 验证结果
-- SELECT column_name, data_type
-- FROM information_schema.columns
-- WHERE table_name = 'images' AND column_name IN ('location', 'latitude', 'longitude');
