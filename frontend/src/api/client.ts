// 1. 对应后端的统一响应结构体
export interface BaseResp<T = any> {
    Code: number;
    Message: string;
    Data: T;
}

// 统一的业务错误类
export class BusinessError extends Error {
    constructor(public code: number, message: string) {
        super(message);
        this.name = 'BusinessError';
    }
}

// 2. 统一请求函数
export async function request<T>(url: string, options?: RequestInit): Promise<T> {
    const response = await fetch(url, {
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...options?.headers,
        },
    });

    // 网络或 HTTP 状态码错误拦截（如 404, 500）
    if (!response.ok) {
        throw new Error(`Network response was not ok: ${response.status}`);
    }

    // 解析出后端统一的 BaseResp 结构
    const result: BaseResp<T> = await response.json();

    // 统一业务状态码拦截（假设 0 代表成功）
    if (result.Code !== 0) {
        // 这里可以加入全局的错误提示逻辑，例如：Toast(result.Message)
        throw new BusinessError(result.Code, result.Message);
    }

    // 成功则直接返回 Data 核心数据
    return result.Data;
}