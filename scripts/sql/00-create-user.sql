-- Create catalogrw role if not exists
DO
$$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'catalogrw') THEN
      CREATE ROLE catalogrw;
   END IF;
   IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'catalog_api') THEN
      CREATE USER catalog_api WITH PASSWORD 'abc@123';
      GRANT catalogrw TO catalog_api;
   END IF;
END
$$;
