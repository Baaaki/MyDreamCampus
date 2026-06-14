import * as SecureStore from "expo-secure-store";

jest.mock("axios", () => {
  // Create a brand-new mock instance every `axios.create()` call so each
  // require()'d copy of api.ts gets its own interceptor registry.
  const make = () => ({
    interceptors: {
      request: { use: jest.fn() },
      response: { use: jest.fn() },
    },
    request: jest.fn(),
    defaults: { headers: {} },
  });
  const mod = {
    __esModule: true,
    default: { create: jest.fn(make), post: jest.fn() },
    AxiosError: class AxiosError extends Error {},
    create: jest.fn(make),
    post: jest.fn(),
  };
  // Keep `default.create` and top-level `create` referring to the same mock so
  // tests can inspect either path.
  mod.default.create = mod.create;
  mod.default.post = mod.post;
  return mod;
});

import axios from "axios";

type AnyMock = { mock: { calls: unknown[][]; results: { value: unknown }[] } };
const axiosMock = axios as unknown as {
  create: jest.Mock;
  post: jest.Mock;
};

type MockApiInstance = {
  interceptors: {
    request: { use: AnyMock & jest.Mock };
    response: { use: AnyMock & jest.Mock };
  };
  request: jest.Mock;
};

function loadApi(): { api: MockApiInstance; mod: typeof import("./api") } {
  let mod!: typeof import("./api");
  jest.isolateModules(() => {
    mod = require("./api");
  });
  // The latest axios.create() result is the freshly-created instance.
  const last =
    axiosMock.create.mock.results[axiosMock.create.mock.results.length - 1]
      ?.value as MockApiInstance;
  return { api: last, mod };
}

beforeEach(() => {
  (SecureStore as unknown as { __resetStore: () => void }).__resetStore();
  jest.clearAllMocks();
});

describe("api.ts request interceptor", () => {
  it("attaches Bearer token from SecureStore when present", async () => {
    await SecureStore.setItemAsync("jwt_token", "secret-jwt-123");
    const { api } = loadApi();

    const requestInterceptor = api.interceptors.request.use.mock.calls[0][0] as (
      c: { headers: Record<string, string> }
    ) => Promise<{ headers: Record<string, string> }>;
    const result = await requestInterceptor({ headers: {} });
    expect(result.headers.Authorization).toBe("Bearer secret-jwt-123");
  });

  it("leaves headers untouched when no token in SecureStore", async () => {
    const { api } = loadApi();
    const requestInterceptor = api.interceptors.request.use.mock.calls[0][0] as (
      c: { headers: Record<string, string> }
    ) => Promise<{ headers: Record<string, string> }>;
    const result = await requestInterceptor({ headers: {} });
    expect(result.headers.Authorization).toBeUndefined();
  });
});

describe("api.ts response interceptor (401 + refresh)", () => {
  type ErrHandler = (err: unknown) => Promise<unknown>;
  function getErrHandler(api: MockApiInstance): ErrHandler {
    return api.interceptors.response.use.mock.calls[0][1] as ErrHandler;
  }

  it("propagates non-401 errors unchanged", async () => {
    const { api } = loadApi();
    const err = { response: { status: 500 }, config: { url: "/x" } };
    await expect(getErrHandler(api)(err)).rejects.toBe(err);
  });

  it("does not refresh on /auth/login 401", async () => {
    const { api } = loadApi();
    const err = { response: { status: 401 }, config: { url: "/auth/login" } };
    await expect(getErrHandler(api)(err)).rejects.toBe(err);
    expect(axiosMock.post).not.toHaveBeenCalled();
  });

  it("does not refresh on /auth/refresh 401", async () => {
    const { api } = loadApi();
    const err = { response: { status: 401 }, config: { url: "/auth/refresh" } };
    await expect(getErrHandler(api)(err)).rejects.toBe(err);
    expect(axiosMock.post).not.toHaveBeenCalled();
  });

  it("clears SecureStore + invokes onUnauthorized when no refresh_token", async () => {
    const { api, mod } = loadApi();
    let calls = 0;
    mod.setOnUnauthorized(() => {
      calls++;
    });

    const err = { response: { status: 401 }, config: { url: "/students/me" } };
    await expect(getErrHandler(api)(err)).rejects.toBe(err);

    expect(calls).toBe(1);
    expect(await SecureStore.getItemAsync("jwt_token")).toBeNull();
    expect(await SecureStore.getItemAsync("refresh_token")).toBeNull();
  });

  it("retries the original request once after successful refresh", async () => {
    await SecureStore.setItemAsync("refresh_token", "rt-1");

    axiosMock.post.mockResolvedValueOnce({
      data: { access_token: "new-at", refresh_token: "new-rt" },
    });

    const { api } = loadApi();
    api.request.mockResolvedValueOnce({ data: { ok: true } });

    const err = {
      response: { status: 401 },
      config: { url: "/students/me", headers: {} as Record<string, string> },
    };
    const result = await getErrHandler(api)(err);
    expect(result).toEqual({ data: { ok: true } });

    const replayConfig = api.request.mock.calls[0][0] as {
      headers: Record<string, string>;
      _retried?: boolean;
    };
    expect(replayConfig.headers.Authorization).toBe("Bearer new-at");
    expect(replayConfig._retried).toBe(true);

    expect(await SecureStore.getItemAsync("jwt_token")).toBe("new-at");
    expect(await SecureStore.getItemAsync("refresh_token")).toBe("new-rt");
  });

  it("does not loop: retried requests bail out and trigger logout", async () => {
    await SecureStore.setItemAsync("jwt_token", "expired");

    const { api, mod } = loadApi();
    let unauthorized = 0;
    mod.setOnUnauthorized(() => {
      unauthorized++;
    });

    const err = {
      response: { status: 401 },
      config: { url: "/students/me", _retried: true },
    };
    await expect(getErrHandler(api)(err)).rejects.toBe(err);

    expect(unauthorized).toBe(1);
    expect(await SecureStore.getItemAsync("jwt_token")).toBeNull();
  });
});
