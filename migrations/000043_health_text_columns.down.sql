ALTER TABLE health_exam_result_items ALTER COLUMN reference_text TYPE VARCHAR(255) USING left(reference_text, 255);
ALTER TABLE health_exam_result_items ALTER COLUMN method TYPE VARCHAR(100) USING left(method, 100);
ALTER TABLE health_exam_result_items ALTER COLUMN material TYPE VARCHAR(100) USING left(material, 100);
ALTER TABLE health_markers ALTER COLUMN default_ref_text TYPE VARCHAR(255) USING left(default_ref_text, 255);
