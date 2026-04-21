"use client";

import { useState, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { motion } from "framer-motion";
import {
  Search, Plus, Star, Package, RefreshCw, Trash2, Edit,
  Upload, X, ShoppingCart, ImageIcon,
} from "lucide-react";
import { Header } from "@/components/layout/header";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { productsApi, type CreateProductRequest, type Product } from "@/lib/api";
import { formatCurrency, getImageUrl } from "@/lib/utils";
import { useAuthStore, useCartStore } from "@/store/useStore";

type ProductForm = CreateProductRequest & { tags_str?: string };

const emptyForm: ProductForm = {
  name: "", description: "", price: 0, category: "", sku: "", stock: 0, tags_str: "",
};

function ProductModal({
  title,
  form,
  setForm,
  onClose,
  onSubmit,
  isPending,
  errorMsg,
  productId,
}: {
  title: string;
  form: ProductForm;
  setForm: (f: ProductForm) => void;
  onClose: () => void;
  onSubmit: () => void;
  isPending: boolean;
  errorMsg: string;
  productId?: string;
}) {
  const queryClient = useQueryClient();
  const fileRef = useRef<HTMLInputElement>(null);
  const [uploadError, setUploadError] = useState("");
  const [uploadingImg, setUploadingImg] = useState(false);

  const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file || !productId) return;
    setUploadingImg(true);
    setUploadError("");
    try {
      await productsApi.uploadImage(productId, file);
      queryClient.invalidateQueries({ queryKey: ["products"] });
    } catch {
      setUploadError("Не удалось загрузить изображение");
    } finally {
      setUploadingImg(false);
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4"
      onClick={onClose}
    >
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        className="bg-card border border-border rounded-xl p-6 w-full max-w-md shadow-xl max-h-[90vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">{title}</h2>
          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={onClose}>
            <X className="w-4 h-4" />
          </Button>
        </div>

        <div className="space-y-3">
          {(["name", "description", "sku", "category"] as const).map((field) => (
            <div key={field}>
              <Label className="text-xs mb-1 block">
                {field === "name" ? "Название" : field === "description" ? "Описание" : field === "sku" ? "Артикул (SKU)" : "Категория"}
              </Label>
              <Input
                value={form[field] as string}
                onChange={(e) => setForm({ ...form, [field]: e.target.value })}
                placeholder={field === "sku" ? "BRAND-MODEL-COLOR-SIZE" : ""}
              />
            </div>
          ))}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <Label className="text-xs mb-1 block">Цена (₽)</Label>
              <Input
                type="number"
                min={0}
                value={form.price}
                onChange={(e) => setForm({ ...form, price: parseFloat(e.target.value) || 0 })}
              />
            </div>
            <div>
              <Label className="text-xs mb-1 block">Количество</Label>
              <Input
                type="number"
                min={0}
                value={form.stock}
                onChange={(e) => setForm({ ...form, stock: parseInt(e.target.value) || 0 })}
              />
            </div>
          </div>
          <div>
            <Label className="text-xs mb-1 block">Теги (через запятую)</Label>
            <Input
              value={form.tags_str}
              onChange={(e) => setForm({ ...form, tags_str: e.target.value })}
              placeholder="электроника, смартфон, ..."
            />
          </div>

          {productId && (
            <div>
              <Label className="text-xs mb-1 block">Изображение товара</Label>
              <input
                ref={fileRef}
                type="file"
                accept="image/*"
                className="hidden"
                onChange={handleImageUpload}
              />
              <Button
                type="button"
                variant="outline"
                className="w-full gap-2 h-9"
                disabled={uploadingImg}
                onClick={() => fileRef.current?.click()}
              >
                <Upload className="w-3.5 h-3.5" />
                {uploadingImg ? "Загрузка..." : "Загрузить фото"}
              </Button>
              {uploadError && (
                <p className="text-xs text-destructive mt-1">{uploadError}</p>
              )}
            </div>
          )}
        </div>

        {errorMsg && (
          <div className="mt-3 text-xs text-destructive bg-destructive/10 rounded-md px-3 py-2">
            {errorMsg}
          </div>
        )}

        <div className="flex gap-3 mt-5">
          <Button variant="outline" className="flex-1" onClick={onClose}>
            Отмена
          </Button>
          <Button className="flex-1" disabled={isPending} onClick={onSubmit}>
            {isPending ? "Сохранение..." : "Сохранить"}
          </Button>
        </div>
      </motion.div>
    </div>
  );
}

export default function ProductsPage() {
  const queryClient = useQueryClient();
  const { user } = useAuthStore();
  const { addItem } = useCartStore();
  const isAdmin = user?.role === "admin";

  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [editProduct, setEditProduct] = useState<Product | null>(null);
  const [createForm, setCreateForm] = useState<ProductForm>(emptyForm);
  const [editForm, setEditForm] = useState<ProductForm>(emptyForm);
  const [createError, setCreateError] = useState("");
  const [editError, setEditError] = useState("");
  const [createdProductId, setCreatedProductId] = useState<string | undefined>();

  const { data, isLoading } = useQuery({
    queryKey: ["products", { page, search, category }],
    queryFn: () =>
      search
        ? productsApi.search({ q: search, category, page, limit: 12 })
        : productsApi.list({ page, limit: 12, category }),
    select: (res) => res.data,
  });

  const parseForm = (f: ProductForm): CreateProductRequest => ({
    name: f.name,
    description: f.description,
    price: f.price,
    category: f.category,
    sku: f.sku,
    stock: f.stock,
    tags: f.tags_str ? f.tags_str.split(",").map((t) => t.trim()).filter(Boolean) : [],
  });

  const createMutation = useMutation({
    mutationFn: (data: CreateProductRequest) => productsApi.create(data),
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: ["products"] });
      setCreatedProductId(res.data.data?.id);
      setCreateError("");
    },
    onError: (err: unknown) => {
      const axiosErr = err as { response?: { data?: { error?: string; message?: string } } };
      const msg =
        axiosErr?.response?.data?.error ||
        axiosErr?.response?.data?.message ||
        "Ошибка при создании товара";
      setCreateError(msg);
    },
  });

  const editMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<CreateProductRequest> }) =>
      productsApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["products"] });
      setEditProduct(null);
      setEditError("");
    },
    onError: (err: unknown) => {
      const axiosErr = err as { response?: { data?: { error?: string; message?: string } } };
      const msg =
        axiosErr?.response?.data?.error ||
        axiosErr?.response?.data?.message ||
        "Ошибка при сохранении товара";
      setEditError(msg);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: productsApi.delete,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["products"] }),
  });

  const handleCreateSubmit = () => {
    setCreateError("");
    createMutation.mutate(parseForm(createForm));
  };

  const handleEditSubmit = () => {
    if (!editProduct) return;
    setEditError("");
    editMutation.mutate({ id: editProduct.id, data: parseForm(editForm) });
  };

  const openEdit = (product: Product) => {
    setEditProduct(product);
    setEditForm({
      name: product.name,
      description: product.description,
      price: product.price,
      category: product.category,
      sku: product.sku,
      stock: product.stock,
      tags_str: (product.tags ?? []).join(", "),
    });
    setEditError("");
  };

  const handleCloseCreate = () => {
    setShowCreate(false);
    setCreateForm(emptyForm);
    setCreateError("");
    setCreatedProductId(undefined);
    createMutation.reset();
  };

  const products = data?.data ?? [];
  const meta = data?.meta;

  return (
    <div className="flex flex-col">
      <Header title="Товары" description={`Всего: ${meta?.total ?? 0}`} />
      <div className="p-6 space-y-4">
        <div className="flex gap-3">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input
              placeholder="Поиск товаров..."
              className="pl-9"
              value={search}
              onChange={(e) => { setSearch(e.target.value); setPage(1); }}
            />
          </div>
          <Input
            placeholder="Категория"
            className="w-40"
            value={category}
            onChange={(e) => { setCategory(e.target.value); setPage(1); }}
          />
          {isAdmin && (
            <Button onClick={() => setShowCreate(true)} className="gap-2">
              <Plus className="w-4 h-4" /> Добавить
            </Button>
          )}
        </div>

        {isLoading ? (
          <div className="flex items-center justify-center h-60">
            <RefreshCw className="w-6 h-6 animate-spin text-muted-foreground" />
          </div>
        ) : products.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-60 text-muted-foreground">
            <Package className="w-12 h-12 mb-3 opacity-30" />
            <p>Товары не найдены</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {products.map((product, i) => (
              <motion.div
                key={product.id}
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ delay: i * 0.04 }}
              >
                <Card className="group overflow-hidden hover:shadow-lg transition-all duration-200 hover:-translate-y-0.5 flex flex-col">
                  <div className="aspect-square bg-muted/50 relative overflow-hidden">
                    {product.images && product.images.length > 0 && getImageUrl(product.images[0]) ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img
                        src={getImageUrl(product.images[0])!}
                        alt={product.name}
                        className="object-cover w-full h-full"
                        onError={(e) => { (e.target as HTMLImageElement).style.display = "none"; }}
                      />
                    ) : (
                      <div className="flex items-center justify-center h-full">
                        <Package className="w-12 h-12 text-muted-foreground/30" />
                      </div>
                    )}
                    <div className="absolute top-2 right-2 flex gap-1">
                      {!product.is_active && (
                        <Badge variant="destructive" className="text-xs">Неакт.</Badge>
                      )}
                    </div>
                    {isAdmin && (
                      <div className="absolute inset-0 bg-black/60 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-2">
                        <Button
                          size="icon"
                          variant="secondary"
                          className="h-8 w-8"
                          onClick={() => openEdit(product)}
                        >
                          <Edit className="w-3.5 h-3.5" />
                        </Button>
                        <Button
                          size="icon"
                          variant="destructive"
                          className="h-8 w-8"
                          onClick={() => deleteMutation.mutate(product.id)}
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </Button>
                      </div>
                    )}
                  </div>
                  <CardContent className="p-4 flex-1">
                    <div className="space-y-1">
                      <p className="text-xs text-muted-foreground uppercase tracking-wider">{product.category}</p>
                      <h3 className="font-semibold text-sm leading-tight line-clamp-2">{product.name}</h3>
                      <p className="text-xs text-muted-foreground line-clamp-2">{product.description}</p>
                    </div>
                  </CardContent>
                  <CardFooter className="p-4 pt-0 flex items-center justify-between">
                    <div>
                      <p className="text-lg font-bold">{formatCurrency(product.price)}</p>
                      <p className="text-xs text-muted-foreground">Склад: {product.stock} шт.</p>
                    </div>
                    <div className="flex items-center gap-2">
                      <div className="flex items-center gap-1">
                        <Star className="w-3.5 h-3.5 fill-yellow-400 text-yellow-400" />
                        <span className="text-xs font-medium">{product.rating?.toFixed(1) ?? "0.0"}</span>
                      </div>
                      {!isAdmin && product.stock > 0 && (
                        <Button
                          size="icon"
                          variant="outline"
                          className="h-7 w-7"
                          title="В корзину"
                          onClick={() => addItem(product)}
                        >
                          <ShoppingCart className="w-3.5 h-3.5" />
                        </Button>
                      )}
                    </div>
                  </CardFooter>
                </Card>
              </motion.div>
            ))}
          </div>
        )}

        {meta && meta.pages > 1 && (
          <div className="flex items-center justify-center gap-2">
            <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
              Назад
            </Button>
            <span className="text-sm text-muted-foreground">{page} / {meta.pages}</span>
            <Button variant="outline" size="sm" disabled={page >= meta.pages} onClick={() => setPage((p) => p + 1)}>
              Вперёд
            </Button>
          </div>
        )}

        {showCreate && (
          createdProductId ? (
            <div
              className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4"
              onClick={handleCloseCreate}
            >
              <motion.div
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{ opacity: 1, scale: 1 }}
                className="bg-card border border-border rounded-xl p-6 w-full max-w-md shadow-xl"
                onClick={(e) => e.stopPropagation()}
              >
                <div className="flex items-center justify-between mb-4">
                  <h2 className="text-lg font-semibold">Товар создан!</h2>
                  <Button variant="ghost" size="icon" className="h-7 w-7" onClick={handleCloseCreate}>
                    <X className="w-4 h-4" />
                  </Button>
                </div>
                <p className="text-sm text-muted-foreground mb-4">
                  Хотите добавить изображение к созданному товару?
                </p>
                <ImageUploadStep productId={createdProductId} onDone={handleCloseCreate} />
              </motion.div>
            </div>
          ) : (
            <ProductModal
              title="Новый товар"
              form={createForm}
              setForm={setCreateForm}
              onClose={handleCloseCreate}
              onSubmit={handleCreateSubmit}
              isPending={createMutation.isPending}
              errorMsg={createError}
            />
          )
        )}

        {editProduct && (
          <ProductModal
            title="Редактировать товар"
            form={editForm}
            setForm={setEditForm}
            onClose={() => { setEditProduct(null); setEditError(""); }}
            onSubmit={handleEditSubmit}
            isPending={editMutation.isPending}
            errorMsg={editError}
            productId={editProduct.id}
          />
        )}
      </div>
    </div>
  );
}

function ImageUploadStep({ productId, onDone }: { productId: string; onDone: () => void }) {
  const queryClient = useQueryClient();
  const fileRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState("");
  const [uploaded, setUploaded] = useState(false);

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    setUploadError("");
    try {
      await productsApi.uploadImage(productId, file);
      queryClient.invalidateQueries({ queryKey: ["products"] });
      setUploaded(true);
    } catch {
      setUploadError("Не удалось загрузить изображение");
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="space-y-3">
      {!uploaded ? (
        <>
          <input ref={fileRef} type="file" accept="image/*" className="hidden" onChange={handleUpload} />
          <Button
            variant="outline"
            className="w-full gap-2"
            disabled={uploading}
            onClick={() => fileRef.current?.click()}
          >
            <ImageIcon className="w-4 h-4" />
            {uploading ? "Загрузка..." : "Выбрать изображение"}
          </Button>
          {uploadError && <p className="text-xs text-destructive">{uploadError}</p>}
        </>
      ) : (
        <p className="text-sm text-green-500 font-medium text-center">Изображение загружено ✓</p>
      )}
      <Button variant={uploaded ? "default" : "ghost"} className="w-full" onClick={onDone}>
        {uploaded ? "Готово" : "Пропустить"}
      </Button>
    </div>
  );
}
