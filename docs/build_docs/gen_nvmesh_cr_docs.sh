
api_dir=../../api/v1
output_file=../nvmesh_cr_api_ref.md
gen_tool=./crd-docs-generator
$gen_tool -v 1 -config ./config.json -api-dir $api_dir -template-dir templates/markdown -out-file $output_file


