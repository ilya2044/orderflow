"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { motion } from "framer-motion";
import { Search, Plus, Star, Package, RefreshCw, Trash2, Edit } from "lucide-react";
import { Header } from "@/components/layout/header";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { productsApi, type CreateProductRequest } from "@/lib/api";
import { formatCurrency } from "@/lib/utils";
import { useAuthStore } from "@/store/useStore";

export default function ProductsPage() {
  const queryClient = useQueryClient();
  const { user } = useAuthStore();
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState<CreateProductRequest>({
    name: "", description: "", price: 0, category: "", sku: "", stock: 0,
  });

  const { data, isLoading } = useQuery({
    queryKey: ["products", { page, search, category }],
    queryFn: () =>
      search
        ? productsApi.search({ q: search, category, page, limit: 12 })
        : productsApi.list({ page, limit: 12, category }),
    select: (res) => res.data,
  });

  const createMutation = useMutation({
    mutationFn: productsApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["products"] });
      setShowCreate(false);
      setForm({ name: "", description: "", price: 0, category: "", sku: "", stock: 0 });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: productsApi.delete,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["products"] }),
  });

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
          {user?.role === "admin" && (
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
                <Card className="group overflow-hidden hover:shadow-lg transition-all duration-200 hover:-translate-y-0.5">
                  <div className="aspect-square bg-muted/50 relative overflow-hidden">
                    {product.images && product.images.length > 0 ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img
                        src={product.images[0]}
                        alt={product.name}
                        className="object-cover w-full h-full"
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
                    {user?.role === "admin" && (
                      <div className="absolute inset-0 bg-black/60 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-2">
                        <Button size="icon" variant="secondary" className="h-8 w-8">
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
                  <CardContent className="p-4">
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
                    <div className="flex items-center gap-1">
                      <Star className="w-3.5 h-3.5 fill-yellow-400 text-yellow-400" />
                      <span className="text-xs font-medium">{product.rating?.toFixed(1) ?? "0.0"}</span>
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
          <div
            className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4"
            onClick={() => setShowCreate(false)}
          >
            <motion.div
              initial={{ opacity: 0, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              className="bg-card border border-border rounded-xl p-6 w-full max-w-md shadow-xl"
              onClick={(e) => e.stopPropagation()}
            >
              <h2 className="text-lg font-semibold mb-4">Новый товар</h2>
              <div className="space-y-3">
                {(["name", "description", "sku", "category"] as const).map((field) => (
                  <div key={field}>
                    <Label className="text-xs capitalize mb-1 block">
                      {field === "name" ? "Название" : field === "description" ? "Описание" : field === "sku" ? "Артикул" : "Категория"}
                    </Label>
                    <Input
                      value={form[field] as string}
                      onChange={(e) => setForm({ ...form, [field]: e.target.value })}
                    />
                  </div>
                ))}
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <Label className="text-xs mb-1 block">Цена (₽)</Label>
                    <Input
                      type="number"
                      value={form.price}
                      onChange={(e) => setForm({ ...form, price: parseFloat(e.target.value) })}
                    />
                  </div>
                  <div>
                    <Label className="text-xs mb-1 block">Количество</Label>
                    <Input
                      type="number"
                      value={form.stock}
                      onChange={(e) => setForm({ ...form, stock: parseInt(e.target.value) })}
                    />
                  </div>
                </div>
              </div>
              <div className="flex gap-3 mt-5">
                <Button variant="outline" className="flex-1" onClick={() => setShowCreate(false)}>
                  Отмена
                </Button>
                <Button
                  className="flex-1"
                  disabled={createMutation.isPending}
                  onClick={() => createMutation.mutate(form)}
                >
                  {createMutation.isPending ? "Создание..." : "Создать"}
                </Button>
              </div>
            </motion.div>
          </div>
        )}
      </div>
    </div>
  );
}
