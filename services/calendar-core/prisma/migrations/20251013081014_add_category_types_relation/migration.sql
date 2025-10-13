-- CreateTable: Tabela de relacionamento many-to-many entre Category e CategoryType
CREATE TABLE "category_to_types" (
    "id" TEXT NOT NULL,
    "category_id" TEXT NOT NULL,
    "category_type_id" TEXT NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "category_to_types_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "category_to_types_category_id_idx" ON "category_to_types"("category_id");

-- CreateIndex
CREATE INDEX "category_to_types_category_type_id_idx" ON "category_to_types"("category_type_id");

-- CreateIndex: Garantir que não haja duplicatas
CREATE UNIQUE INDEX "category_to_types_category_id_category_type_id_key" ON "category_to_types"("category_id", "category_type_id");

-- AddForeignKey
ALTER TABLE "category_to_types" ADD CONSTRAINT "category_to_types_category_id_fkey" FOREIGN KEY ("category_id") REFERENCES "categories"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "category_to_types" ADD CONSTRAINT "category_to_types_category_type_id_fkey" FOREIGN KEY ("category_type_id") REFERENCES "category_types"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- Migrar dados existentes: converter type string para relacionamento
INSERT INTO "category_to_types" ("id", "category_id", "category_type_id", "created_at")
SELECT 
    gen_random_uuid(),
    c.id,
    ct.id,
    CURRENT_TIMESTAMP
FROM "categories" c
INNER JOIN "category_types" ct ON ct.value = c.type
WHERE c.type IS NOT NULL AND c.type != '';
