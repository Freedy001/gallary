-- PostgreSQL 数据库表结构
-- 图片管理系统

-- 图片表
CREATE TABLE IF NOT EXISTS images (
    id BIGSERIAL PRIMARY KEY,
    uuid VARCHAR(36) UNIQUE NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    storage_path VARCHAR(500) NOT NULL,
    storage_type VARCHAR(20) NOT NULL DEFAULT 'local',
    file_size BIGINT NOT NULL,
    file_hash VARCHAR(64) UNIQUE NOT NULL, -- SHA256 哈希值用于去重
    mime_type VARCHAR(50) NOT NULL,
    width INTEGER,
    height INTEGER,

    -- EXIF 元数据
    taken_at TIMESTAMP,
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    location_name VARCHAR(255),
    camera_model VARCHAR(100),
    camera_make VARCHAR(100),
    aperture VARCHAR(20),
    shutter_speed VARCHAR(20),
    iso INTEGER,
    focal_length VARCHAR(20),

    -- 系统字段
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_images_taken_at ON images(taken_at);
CREATE INDEX idx_images_location ON images(latitude, longitude) WHERE latitude IS NOT NULL AND longitude IS NOT NULL;
CREATE INDEX idx_images_created_at ON images(created_at);
CREATE INDEX idx_images_deleted_at ON images(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_images_file_hash ON images(file_hash);

COMMENT ON TABLE images IS '图片表';
COMMENT ON COLUMN images.file_hash IS 'SHA256哈希值用于去重';

-- 标签表
CREATE TABLE IF NOT EXISTS tags (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    color VARCHAR(7),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tags_name ON tags(name);

COMMENT ON TABLE tags IS '标签表';

-- 图片标签关联表
CREATE TABLE IF NOT EXISTS image_tags (
    id BIGSERIAL PRIMARY KEY,
    image_id BIGINT NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    tag_id BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(image_id, tag_id)
);

CREATE INDEX idx_image_tags_image_id ON image_tags(image_id);
CREATE INDEX idx_image_tags_tag_id ON image_tags(tag_id);

COMMENT ON TABLE image_tags IS '图片标签关联表';

-- 自定义元数据表
CREATE TABLE IF NOT EXISTS image_metadata (
    id BIGSERIAL PRIMARY KEY,
    image_id BIGINT NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    meta_key VARCHAR(100) NOT NULL,
    meta_value TEXT,
    value_type VARCHAR(20) DEFAULT 'string',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(image_id, meta_key)
);

CREATE INDEX idx_image_metadata_image_id ON image_metadata(image_id);
CREATE INDEX idx_image_metadata_meta_key ON image_metadata(meta_key);

COMMENT ON TABLE image_metadata IS '图片自定义元数据表';
COMMENT ON COLUMN image_metadata.value_type IS '值类型: string/number/boolean/json';

-- 分享表
CREATE TABLE IF NOT EXISTS shares (
    id BIGSERIAL PRIMARY KEY,
    share_code VARCHAR(32) UNIQUE NOT NULL,
    title VARCHAR(255),
    description TEXT,
    password VARCHAR(64),
    expire_at TIMESTAMP,
    view_count INTEGER DEFAULT 0,
    download_count INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_shares_share_code ON shares(share_code);
CREATE INDEX idx_shares_expire_at ON shares(expire_at);
CREATE INDEX idx_shares_is_active ON shares(is_active);

COMMENT ON TABLE shares IS '分享表';

-- 分享图片关联表
CREATE TABLE IF NOT EXISTS share_images (
    id BIGSERIAL PRIMARY KEY,
    share_id BIGINT NOT NULL REFERENCES shares(id) ON DELETE CASCADE,
    image_id BIGINT NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(share_id, image_id)
);

CREATE INDEX idx_share_images_share_id ON share_images(share_id);
CREATE INDEX idx_share_images_image_id ON share_images(image_id);

COMMENT ON TABLE share_images IS '分享图片关联表';

-- 系统配置表
CREATE TABLE IF NOT EXISTS system_config (
    id BIGSERIAL PRIMARY KEY,
    config_key VARCHAR(100) UNIQUE NOT NULL,
    config_value TEXT,
    description VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_system_config_key ON system_config(config_key);

COMMENT ON TABLE system_config IS '系统配置表';

-- 创建更新时间触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为所有表添加更新时间触发器
CREATE TRIGGER update_images_updated_at BEFORE UPDATE ON images
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tags_updated_at BEFORE UPDATE ON tags
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_image_metadata_updated_at BEFORE UPDATE ON image_metadata
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_shares_updated_at BEFORE UPDATE ON shares
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_system_config_updated_at BEFORE UPDATE ON system_config
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
