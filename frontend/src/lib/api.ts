import axios, { type AxiosError } from "axios";

export const api = axios.create({
  baseURL: "/api/v1",
  headers: { "Content-Type": "application/json" },
  timeout: 15000,
});

api.interceptors.request.use((config) => {
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("access_token");
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
  }
  return config;
});

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as typeof error.config & { _retry?: boolean };
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;
      try {
        const refreshToken = localStorage.getItem("refresh_token");
        if (!refreshToken) {
          throw new Error("no refresh token");
        }
        const res = await axios.post("/api/v1/auth/refresh", {
          refresh_token: refreshToken,
        });
        const { access_token } = res.data.data;
        localStorage.setItem("access_token", access_token);
        originalRequest.headers!.Authorization = `Bearer ${access_token}`;
        return api(originalRequest);
      } catch {
        localStorage.removeItem("access_token");
        localStorage.removeItem("refresh_token");
        window.location.href = "/login";
      }
    }
    return Promise.reject(error);
  }
);

export interface PaginatedResponse<T> {
  success: boolean;
  data: T[];
  meta: { total: number; page: number; limit: number; pages: number };
}

export interface ApiResponse<T> {
  success: boolean;
  data: T;
  message?: string;
  error?: string;
}

export const authApi = {
  register: (data: { email: string; username: string; password: string }) =>
    api.post<ApiResponse<TokenPair>>("/auth/register", data),
  login: (data: { email: string; password: string }) =>
    api.post<ApiResponse<TokenPair>>("/auth/login", data),
  refresh: (refreshToken: string) =>
    api.post<ApiResponse<TokenPair>>("/auth/refresh", { refresh_token: refreshToken }),
  logout: (refreshToken: string) =>
    api.post("/auth/logout", { refresh_token: refreshToken }),
  me: () => api.get<ApiResponse<User>>("/auth/me"),
};

export const ordersApi = {
  list: (params?: Record<string, unknown>) =>
    api.get<PaginatedResponse<Order>>("/orders", { params }),
  get: (id: string) => api.get<ApiResponse<Order>>(`/orders/${id}`),
  create: (data: CreateOrderRequest) =>
    api.post<ApiResponse<Order>>("/orders", data),
  updateStatus: (id: string, status: string) =>
    api.put<ApiResponse<Order>>(`/orders/${id}/status`, { status }),
  cancel: (id: string) => api.delete(`/orders/${id}`),
};

export const productsApi = {
  list: (params?: Record<string, unknown>) =>
    api.get<PaginatedResponse<Product>>("/products", { params }),
  search: (params?: Record<string, unknown>) =>
    api.get<PaginatedResponse<Product>>("/products/search", { params }),
  get: (id: string) => api.get<ApiResponse<Product>>(`/products/${id}`),
  create: (data: CreateProductRequest) =>
    api.post<ApiResponse<Product>>("/products", data),
  update: (id: string, data: Partial<CreateProductRequest>) =>
    api.put<ApiResponse<Product>>(`/products/${id}`, data),
  delete: (id: string) => api.delete(`/products/${id}`),
  uploadImage: (id: string, file: File) => {
    const form = new FormData();
    form.append("image", file);
    return api.post<ApiResponse<Product>>(`/products/${id}/images`, form, {
      headers: { "Content-Type": "multipart/form-data" },
    });
  },
};

export const usersApi = {
  list: (params?: Record<string, unknown>) =>
    api.get<PaginatedResponse<User>>("/users", { params }),
  get: (id: string) => api.get<ApiResponse<User>>(`/users/${id}`),
  update: (id: string, data: Partial<User>) =>
    api.put<ApiResponse<User>>(`/users/${id}`, data),
  delete: (id: string) => api.delete(`/users/${id}`),
};

export const paymentsApi = {
  list: (params?: Record<string, unknown>) =>
    api.get<ApiResponse<Payment[]>>("/payments", { params }),
  get: (id: string) => api.get<ApiResponse<Payment>>(`/payments/${id}`),
  create: (data: CreatePaymentRequest) =>
    api.post<ApiResponse<Payment>>("/payments", data),
};

export interface Notification {
  id: string;
  user_id: string;
  type: string;
  subject: string;
  body: string;
  channel: string;
  status: string;
  created_at: string;
}

export const notificationsApi = {
  listForUser: (userId: string) =>
    api.get<{ success: boolean; data: Notification[] }>(`/notifications/${userId}`),
  listForAdmin: () =>
    api.get<{ success: boolean; data: Notification[] }>("/notifications/admin"),
};

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_at: string;
  user: User;
}

export interface User {
  id: string;
  email: string;
  username: string;
  first_name: string;
  last_name: string;
  phone?: string;
  address?: string;
  avatar_url?: string;
  role: "user" | "admin";
  is_active: boolean;
  created_at: string;
}

export interface Order {
  id: string;
  user_id: string;
  status: "pending" | "confirmed" | "processing" | "shipped" | "delivered" | "cancelled" | "refunded";
  total_price: number;
  shipping_address: string;
  notes?: string;
  items: OrderItem[];
  created_at: string;
  updated_at: string;
}

export interface OrderItem {
  id: string;
  order_id: string;
  product_id: string;
  product_name: string;
  price: number;
  quantity: number;
}

export interface CreateOrderRequest {
  shipping_address: string;
  notes?: string;
  items: { product_id: string; quantity: number; price: number; name: string }[];
}

export interface Product {
  id: string;
  name: string;
  description: string;
  price: number;
  category: string;
  sku: string;
  stock: number;
  images: string[];
  tags: string[];
  is_active: boolean;
  rating: number;
  review_count: number;
  created_at: string;
}

export interface CreateProductRequest {
  name: string;
  description: string;
  price: number;
  category: string;
  sku: string;
  stock: number;
  tags?: string[];
}

export interface Payment {
  id: string;
  order_id: string;
  user_id: string;
  amount: number;
  status: "pending" | "processing" | "completed" | "failed" | "refunded";
  method: string;
  created_at: string;
}

export interface CreatePaymentRequest {
  order_id: string;
  amount: number;
  method: string;
}
