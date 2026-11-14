package postgres

const searchQuery = `
 WITH allowed AS MATERIALIZED (
  SELECT c.*, d.title document_title, s.title section_title, s.anchor
  FROM chunks c
  JOIN documents d ON d.id=c.document_id
  JOIN sections s ON s.document_id=c.document_id AND s.ordinal=c.section
  WHERE d.team IN ('common',$1)
 ), lexical AS (
  SELECT document_id, ordinal,
   row_number() OVER (
    ORDER BY ts_rank_cd(search,q) DESC, document_id, ordinal
   ) rank
  FROM allowed,
   (SELECT plainto_tsquery('russian',$2) || plainto_tsquery('english',$2) q) q
  WHERE search @@ q
  ORDER BY ts_rank_cd(search,q) DESC, document_id, ordinal
  LIMIT 20
 ), semantic AS (
  SELECT document_id, ordinal,
   row_number() OVER (
    ORDER BY embedding <=> $3::vector, document_id, ordinal
   ) rank
  FROM allowed
  WHERE 1-(embedding <=> $3::vector)>=$4
  ORDER BY embedding <=> $3::vector, document_id, ordinal
  LIMIT 20
 ), fused AS (
  SELECT document_id, ordinal, sum(1.0/(60+rank)) score
  FROM (
   SELECT * FROM lexical
   UNION ALL
   SELECT * FROM semantic
  ) candidates
  GROUP BY document_id, ordinal
 )
 SELECT a.document_id, a.document_title, a.section_title, a.anchor, a.text
 FROM fused f
 JOIN allowed a USING(document_id,ordinal)
 ORDER BY f.score DESC, a.document_id, a.ordinal
 LIMIT 3
`
