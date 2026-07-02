-- name: LinkPackagePlatform :exec
INSERT OR IGNORE INTO package_platforms (
    package_id,
    platform_id
) VALUES (
    ?,
    ?
);

-- name: ListPackagePlatforms :many
SELECT platforms.* FROM platforms
JOIN package_platforms ON package_platforms.platform_id = platforms.id
WHERE package_platforms.package_id = ?
ORDER BY platforms.name;

-- name: ListPackagesByPlatform :many
SELECT packages.* FROM packages
JOIN package_platforms ON package_platforms.package_id = packages.id
WHERE package_platforms.platform_id = ?
ORDER BY packages.name, packages.attr;
