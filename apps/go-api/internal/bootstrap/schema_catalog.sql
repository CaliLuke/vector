WITH definitions AS (
  SELECT 'table:' || c.relname AS key, c.relkind::text AS value
  FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
  WHERE n.nspname='public' AND c.relkind IN ('r','p')
  UNION ALL
  SELECT 'column:' || c.relname || '.' || a.attname,
    jsonb_build_array(pg_catalog.format_type(a.atttypid,a.atttypmod),a.attnotnull,
      pg_catalog.pg_get_expr(d.adbin,d.adrelid),a.attidentity,a.attgenerated)::text
  FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
  JOIN pg_catalog.pg_attribute a ON a.attrelid=c.oid
  LEFT JOIN pg_catalog.pg_attrdef d ON d.adrelid=c.oid AND d.adnum=a.attnum
  WHERE n.nspname='public' AND c.relkind IN ('r','p') AND a.attnum>0 AND NOT a.attisdropped
  UNION ALL
  SELECT 'constraint:' || c.relname || '.' || con.conname,
    jsonb_build_array(pg_catalog.pg_get_constraintdef(con.oid),con.convalidated)::text
  FROM pg_catalog.pg_constraint con JOIN pg_catalog.pg_class c ON c.oid=con.conrelid
  JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public'
  UNION ALL
  SELECT 'index:' || c.relname || '.' || i.relname,
    jsonb_build_array(pg_catalog.pg_get_indexdef(x.indexrelid),x.indisvalid,x.indisready)::text
  FROM pg_catalog.pg_index x JOIN pg_catalog.pg_class c ON c.oid=x.indrelid
  JOIN pg_catalog.pg_class i ON i.oid=x.indexrelid
  JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public'
  UNION ALL
  SELECT 'sequence:' || c.relname,
    jsonb_build_array(pg_catalog.format_type(s.seqtypid,NULL),s.seqstart,s.seqincrement,
      s.seqmax,s.seqmin,s.seqcache,s.seqcycle)::text
  FROM pg_catalog.pg_sequence s JOIN pg_catalog.pg_class c ON c.oid=s.seqrelid
  JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public'
)
SELECT COALESCE(jsonb_object_agg(key,value),'{}'::jsonb)::text FROM definitions;
