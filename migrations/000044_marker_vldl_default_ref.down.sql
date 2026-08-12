UPDATE health_markers
SET default_ref_max = NULL,
    default_ref_text = NULL
WHERE scope = 'system'
  AND normalized_key = 'colesterol vldl'
  AND default_ref_max = 30;
