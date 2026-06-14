// Mock the api module before importing authService
jest.mock("./api", () => {
  return {
    __esModule: true,
    default: {
      post: jest.fn(),
    },
  };
});

import * as SecureStore from "expo-secure-store";
import api from "./api";
import authService from "./authService";

const apiMock = api as unknown as { post: jest.Mock };

beforeEach(() => {
  (SecureStore as unknown as { __resetStore: () => void }).__resetStore();
  jest.clearAllMocks();
});

describe("authService.login", () => {
  it("persists access_token, refresh_token, and user on success", async () => {
    apiMock.post.mockResolvedValueOnce({
      data: {
        access_token: "at-1",
        refresh_token: "rt-1",
        user: { id: "u-1", email: "x@y.tr", role: "student" },
      },
    });

    const data = await authService.login({ email: "x@y.tr", password: "Pa55word" });
    expect(data.access_token).toBe("at-1");
    expect(await SecureStore.getItemAsync("jwt_token")).toBe("at-1");
    expect(await SecureStore.getItemAsync("refresh_token")).toBe("rt-1");
    const stored = await SecureStore.getItemAsync("user_data");
    expect(JSON.parse(stored!)).toEqual({ id: "u-1", email: "x@y.tr", role: "student" });
  });

  it("propagates API errors", async () => {
    apiMock.post.mockRejectedValueOnce(new Error("network down"));
    await expect(
      authService.login({ email: "x", password: "y" })
    ).rejects.toThrow("network down");

    expect(await SecureStore.getItemAsync("jwt_token")).toBeNull();
  });

  it("does not store user when missing in response", async () => {
    apiMock.post.mockResolvedValueOnce({
      data: { access_token: "at", refresh_token: "rt" },
    });
    await authService.login({ email: "x", password: "y" });
    expect(await SecureStore.getItemAsync("user_data")).toBeNull();
  });
});

describe("authService.logout", () => {
  it("clears storage even when API call fails", async () => {
    await SecureStore.setItemAsync("jwt_token", "at");
    await SecureStore.setItemAsync("refresh_token", "rt");
    await SecureStore.setItemAsync("user_data", "{}");

    apiMock.post.mockRejectedValueOnce(new Error("server down"));
    await authService.logout();

    expect(await SecureStore.getItemAsync("jwt_token")).toBeNull();
    expect(await SecureStore.getItemAsync("refresh_token")).toBeNull();
    expect(await SecureStore.getItemAsync("user_data")).toBeNull();
  });

  it("clears storage on successful API call", async () => {
    await SecureStore.setItemAsync("jwt_token", "at");
    apiMock.post.mockResolvedValueOnce({ data: { message: "ok" } });
    await authService.logout();
    expect(await SecureStore.getItemAsync("jwt_token")).toBeNull();
  });
});

describe("authService.changePassword", () => {
  it("rotates tokens after successful change", async () => {
    await SecureStore.setItemAsync("jwt_token", "old-at");
    apiMock.post.mockResolvedValueOnce({
      data: { access_token: "new-at", refresh_token: "new-rt" },
    });

    const res = await authService.changePassword({
      old_password: "OldPass1",
      new_password: "NewPass1",
    });
    expect(res.access_token).toBe("new-at");
    expect(await SecureStore.getItemAsync("jwt_token")).toBe("new-at");
    expect(await SecureStore.getItemAsync("refresh_token")).toBe("new-rt");
  });
});

describe("authService.refresh", () => {
  it("returns null when no stored refresh token", async () => {
    expect(await authService.refresh()).toBeNull();
    expect(apiMock.post).not.toHaveBeenCalled();
  });

  it("returns new access token and persists rotation", async () => {
    await SecureStore.setItemAsync("refresh_token", "rt-old");
    apiMock.post.mockResolvedValueOnce({
      data: { access_token: "rotated-at", refresh_token: "rt-new" },
    });

    const got = await authService.refresh();
    expect(got).toBe("rotated-at");
    expect(await SecureStore.getItemAsync("jwt_token")).toBe("rotated-at");
    expect(await SecureStore.getItemAsync("refresh_token")).toBe("rt-new");
  });

  it("returns null on API failure (does not throw)", async () => {
    await SecureStore.setItemAsync("refresh_token", "rt");
    apiMock.post.mockRejectedValueOnce(new Error("401"));
    expect(await authService.refresh()).toBeNull();
  });
});

describe("authService.getStoredUser", () => {
  it("returns parsed user when present", async () => {
    await SecureStore.setItemAsync("user_data", JSON.stringify({ id: "1", role: "admin" }));
    const u = await authService.getStoredUser();
    expect(u).toEqual({ id: "1", role: "admin" });
  });

  it("returns null when missing", async () => {
    expect(await authService.getStoredUser()).toBeNull();
  });

  it("returns null when stored data is corrupt", async () => {
    await SecureStore.setItemAsync("user_data", "{not json");
    expect(await authService.getStoredUser()).toBeNull();
  });
});

describe("authService.isAuthenticated", () => {
  it("true when jwt_token present", async () => {
    await SecureStore.setItemAsync("jwt_token", "at");
    expect(await authService.isAuthenticated()).toBe(true);
  });
  it("false when missing", async () => {
    expect(await authService.isAuthenticated()).toBe(false);
  });
});

describe("authService.clearAuth", () => {
  it("removes all auth keys", async () => {
    await SecureStore.setItemAsync("jwt_token", "a");
    await SecureStore.setItemAsync("refresh_token", "b");
    await SecureStore.setItemAsync("user_data", "{}");

    await authService.clearAuth();

    expect(await SecureStore.getItemAsync("jwt_token")).toBeNull();
    expect(await SecureStore.getItemAsync("refresh_token")).toBeNull();
    expect(await SecureStore.getItemAsync("user_data")).toBeNull();
  });
});
