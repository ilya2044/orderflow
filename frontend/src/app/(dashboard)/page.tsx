"use client";

import { useQuery } from "@tanstack/react-query";
import { motion } from "framer-motion";
import {
  ShoppingCart,
  Package,
  Users,
  DollarSign,
  ArrowUpRight,
  Clock,
  CheckCircle2,
  Truck,
} from "lucide-react";
import { Header } from "@/components/layout/header";
import { StatsCard } from "@/components/dashboard/stats-card";
import { RevenueChart, OrderStatusChart, CategorySalesChart } from "@/components/dashboard/charts";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { OrderStatusBadge } from "@/components/orders/order-status-badge";
import { ordersApi, productsApi, usersApi, type Order } from "@/lib/api";
import { formatCurrency, formatDate, truncateId } from "@/lib/utils";
import { useAuthStore } from "@/store/useStore";

export default function DashboardPage() {
  const { user } = useAuthStore();
  const isAdmin = user?.role === "admin";

  const { data: ordersData } = useQuery({
    queryKey: ["orders", { limit: 5 }],
    queryFn: () => ordersApi.list({ limit: 5, page: 1 }),
    select: (res) => res.data,
  });

  const { data: allOrdersData } = useQuery({
    queryKey: ["orders-stats"],
    queryFn: () => ordersApi.list({ limit: 1 }),
    select: (res) => res.data,
    enabled: isAdmin,
  });

  const { data: productsData } = useQuery({
    queryKey: ["products-stats"],
    queryFn: () => productsApi.list({ limit: 1 }),
    select: (res) => res.data,
    enabled: isAdmin,
  });

  const { data: usersData } = useQuery({
    queryKey: ["users-stats"],
    queryFn: () => usersApi.list({ limit: 1 }),
    select: (res) => res.data,
    enabled: isAdmin,
  });

  const recentOrders = ordersData?.data ?? [];
  const totalOrders = allOrdersData?.meta.total ?? 0;
  const totalProducts = productsData?.meta.total ?? 0;
  const totalUsers = usersData?.meta.total ?? 0;

  if (!isAdmin) {
    return <UserDashboard recentOrders={recentOrders} totalOrders={ordersData?.meta.total ?? 0} />;
  }

  return (
    <div className="flex flex-col">
      <Header title="Дашборд" description="Обзор системы обработки заказов" />
      <div className="p-6 space-y-6">
        <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4">
          <StatsCard
            title="Выручка (месяц)"
            value={formatCurrency(341000)}
            change={12.5}
            changeLabel="vs прошлый месяц"
            icon={DollarSign}
            iconColor="text-green-500"
            delay={0}
          />
          <StatsCard
            title="Заказы"
            value={totalOrders.toString()}
            change={8.2}
            changeLabel="vs прошлый месяц"
            icon={ShoppingCart}
            iconColor="text-blue-500"
            delay={0.1}
          />
          <StatsCard
            title="Товары"
            value={totalProducts.toString()}
            change={3.1}
            changeLabel="новых"
            icon={Package}
            iconColor="text-purple-500"
            delay={0.2}
          />
          <StatsCard
            title="Пользователи"
            value={totalUsers.toString()}
            change={15.3}
            changeLabel="vs прошлый месяц"
            icon={Users}
            iconColor="text-orange-500"
            delay={0.3}
          />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          <RevenueChart />
          <OrderStatusChart />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.5, duration: 0.4 }}
            className="lg:col-span-2"
          >
            <Card>
              <CardHeader className="flex flex-row items-center justify-between">
                <div>
                  <CardTitle>Последние заказы</CardTitle>
                  <CardDescription>Недавно оформленные заказы</CardDescription>
                </div>
                <a
                  href="/orders"
                  className="text-xs text-primary flex items-center gap-1 hover:underline"
                >
                  Все заказы <ArrowUpRight className="w-3 h-3" />
                </a>
              </CardHeader>
              <CardContent>
                <div className="space-y-0">
                  {recentOrders.length === 0 ? (
                    <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                      <Clock className="w-8 h-8 mb-2 opacity-50" />
                      <p className="text-sm">Заказов пока нет</p>
                    </div>
                  ) : (
                    recentOrders.map((order, i) => (
                      <motion.div
                        key={order.id}
                        initial={{ opacity: 0, x: -10 }}
                        animate={{ opacity: 1, x: 0 }}
                        transition={{ delay: 0.6 + i * 0.05 }}
                        className="flex items-center justify-between py-3 border-b border-border last:border-0"
                      >
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
                            <ShoppingCart className="w-3.5 h-3.5 text-primary" />
                          </div>
                          <div>
                            <p className="text-sm font-medium">#{truncateId(order.id)}</p>
                            <p className="text-xs text-muted-foreground">{formatDate(order.created_at)}</p>
                          </div>
                        </div>
                        <div className="flex items-center gap-4">
                          <OrderStatusBadge status={order.status} />
                          <p className="text-sm font-semibold">{formatCurrency(order.total_price)}</p>
                        </div>
                      </motion.div>
                    ))
                  )}
                </div>
              </CardContent>
            </Card>
          </motion.div>

          <CategorySalesChart />
        </div>
      </div>
    </div>
  );
}

function UserDashboard({
  recentOrders,
  totalOrders,
}: {
  recentOrders: Order[];
  totalOrders: number;
}) {
  const activeOrders = recentOrders.filter(
    (o) => !["delivered", "cancelled", "refunded"].includes(o.status)
  );
  const deliveredOrders = recentOrders.filter((o) => o.status === "delivered");

  return (
    <div className="flex flex-col">
      <Header title="Главная" description="Ваши заказы и покупки" />
      <div className="p-6 space-y-6">
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <StatsCard
            title="Всего заказов"
            value={totalOrders.toString()}
            icon={ShoppingCart}
            iconColor="text-blue-500"
            delay={0}
          />
          <StatsCard
            title="Активные заказы"
            value={activeOrders.length.toString()}
            icon={Truck}
            iconColor="text-orange-500"
            delay={0.1}
          />
          <StatsCard
            title="Выполнено"
            value={deliveredOrders.length.toString()}
            icon={CheckCircle2}
            iconColor="text-green-500"
            delay={0.2}
          />
        </div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3 }}
        >
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <div>
                <CardTitle>Мои заказы</CardTitle>
                <CardDescription>Последние 5 заказов</CardDescription>
              </div>
              <a
                href="/orders"
                className="text-xs text-primary flex items-center gap-1 hover:underline"
              >
                Все заказы <ArrowUpRight className="w-3 h-3" />
              </a>
            </CardHeader>
            <CardContent>
              {recentOrders.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-12 text-muted-foreground gap-3">
                  <ShoppingCart className="w-10 h-10 opacity-20" />
                  <p className="text-sm">У вас пока нет заказов</p>
                  <a
                    href="/products"
                    className="text-xs text-primary hover:underline flex items-center gap-1"
                  >
                    Перейти в каталог <ArrowUpRight className="w-3 h-3" />
                  </a>
                </div>
              ) : (
                recentOrders.map((order, i) => (
                  <motion.div
                    key={order.id}
                    initial={{ opacity: 0, x: -10 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: 0.4 + i * 0.05 }}
                    className="flex items-center justify-between py-3 border-b border-border last:border-0"
                  >
                    <div className="flex items-center gap-3">
                      <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
                        <ShoppingCart className="w-3.5 h-3.5 text-primary" />
                      </div>
                      <div>
                        <p className="text-sm font-medium">#{truncateId(order.id)}</p>
                        <p className="text-xs text-muted-foreground">{formatDate(order.created_at)}</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <OrderStatusBadge status={order.status as "pending"} />
                      <p className="text-sm font-semibold">{formatCurrency(order.total_price)}</p>
                    </div>
                  </motion.div>
                ))
              )}
            </CardContent>
          </Card>
        </motion.div>
      </div>
    </div>
  );
}
