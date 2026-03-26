(function (window, document) {
    const basePath = window.__KUZOVOK_BASE_PATH__ || "";

    function buildPath(path) {
        const normalized = !path ? "" : (path.startsWith("/") ? path : `/${path}`);
        return `${basePath}${normalized || ""}`;
    }

    async function api(endpoint, options = {}, hooks = {}) {
        try {
            const headers = { ...(options.headers || {}) };
            if (!(options.body instanceof FormData)) {
                headers["Content-Type"] = headers["Content-Type"] || "application/json";
            } else {
                delete headers["Content-Type"];
            }

            const response = await fetch(`${buildPath("/api")}${endpoint}`, {
                ...options,
                headers,
                credentials: "include"
            });

            const payload = await response.json().catch(() => ({
                success: false,
                message: "Неожиданный ответ сервера"
            }));

            if (payload && typeof payload === "object") {
                payload.status = response.status;
            }

            if (typeof hooks.onResponse === "function") {
                hooks.onResponse(response, payload);
            }

            return payload;
        } catch (error) {
            console.error("Kuzovok API request failed", error);
            return { success: false, status: 0, message: "Ошибка сети" };
        }
    }

    function esc(value) {
        const div = document.createElement("div");
        div.textContent = value == null ? "" : String(value);
        return div.innerHTML;
    }

    function formatDate(value, options = {}) {
        if (!value) {
            return "—";
        }

        const { includeYear = false, includeTime = true } = options;
        return new Date(value).toLocaleString("ru-RU", {
            day: "numeric",
            month: "long",
            ...(includeYear ? { year: "numeric" } : {}),
            ...(includeTime ? { hour: "2-digit", minute: "2-digit" } : {})
        });
    }

    function truncateQuoted(text, maxLength) {
        if (!text) {
            return "";
        }
        if (text.length <= maxLength) {
            return `"${text}"`;
        }
        return `"${text.substring(0, maxLength)}..."`;
    }

    function formatCountdown(milliseconds) {
        const totalSeconds = Math.max(0, Math.floor(milliseconds / 1000));
        const hours = Math.floor(totalSeconds / 3600);
        const minutes = Math.floor((totalSeconds % 3600) / 60);
        const seconds = totalSeconds % 60;
        return [hours, minutes, seconds].map((value) => String(value).padStart(2, "0")).join(":");
    }

    function syncImageCountdowns(root) {
        const scope = root || document;
        const imageNodes = scope.querySelectorAll(".post-image[data-expires-at]");
        imageNodes.forEach((node) => {
            const expiresAt = Date.parse(node.dataset.expiresAt || "");
            if (!Number.isFinite(expiresAt)) {
                node.remove();
                return;
            }

            const remaining = expiresAt - Date.now();
            if (remaining <= 0) {
                node.remove();
                return;
            }

            const timer = node.querySelector(".image-timer");
            if (timer) {
                timer.textContent = formatCountdown(remaining);
            }
        });
    }

    function resolveMediaUrl(url) {
        if (!url) {
            return "";
        }
        if (url.startsWith("http://") || url.startsWith("https://")) {
            return url;
        }
        return `${basePath}${url}`;
    }

    function userUrl(username) {
        return buildPath(`/user/${encodeURIComponent(username)}`);
    }

    function profileUrl() {
        return buildPath("/profile");
    }

    function homeUrl() {
        return buildPath("/");
    }

    function adminUrl() {
        return buildPath("/admin");
    }

    function renderPostImage(post, altText) {
        if (!post.image_url || !post.image_expires_at) {
            return "";
        }

        const alt = altText || "Картинка к посту";
        return `<div class="post-image" data-expires-at="${esc(post.image_expires_at)}"><img src="${esc(resolveMediaUrl(post.image_url))}" alt="${esc(alt)}"><div class="image-timer"></div></div>`;
    }

    function renderParentPost(parentPost, options = {}) {
        if (!parentPost || !parentPost.id) {
            return "";
        }

        const action = options.action || "";
        const actionAttrs = [];
        if (action) {
            actionAttrs.push(`data-parent-action="${esc(action)}"`);
        }
        if (parentPost.id != null) {
            actionAttrs.push(`data-parent-id="${esc(parentPost.id)}"`);
        }
        if (parentPost.username) {
            actionAttrs.push(`data-parent-username="${esc(parentPost.username)}"`);
        }

        return `<div class="post-parent" ${actionAttrs.join(" ")}><span class="post-author-avatar">${esc(parentPost.avatar || "🐠")}</span> ↩ Ответ на: <a href="${userUrl(parentPost.username)}" class="post-parent-link"><strong>@${esc(parentPost.username)}</strong></a> — ${esc(truncateQuoted(parentPost.content, 20))}</div>`;
    }

    function updateLikeButton(button, data) {
        if (!button || !data) {
            return;
        }

        const count = button.querySelector("[data-role='like-count']") || button.querySelector("span");
        if (count) {
            count.textContent = data.likes;
        }
        button.classList.toggle("liked", Boolean(data.liked));
        button.setAttribute("aria-label", data.liked ? "Убрать лайк" : "Поставить лайк");
    }

    function highlightNode(node, duration) {
        if (!node) {
            return;
        }
        node.classList.add("post-highlighted");
        window.setTimeout(() => node.classList.remove("post-highlighted"), duration || 2000);
    }

    function delegate(root, eventName, selector, handler) {
        if (!root) {
            return;
        }
        root.addEventListener(eventName, (event) => {
            const target = event.target.closest(selector);
            if (!target || !root.contains(target)) {
                return;
            }
            handler(event, target);
        });
    }

    function wireModalClose(overlay, onClose) {
        if (!overlay) {
            return;
        }
        overlay.addEventListener("click", (event) => {
            if (event.target === overlay && typeof onClose === "function") {
                onClose();
            }
        });
    }

    function bootOceanScene(options = {}) {
        const container = document.getElementById(options.containerId || "oceanBg");
        if (!container) {
            return;
        }

        container.innerHTML = "";

        const bubbleCount = options.bubbleCount || 10;
        const fishCount = options.fishCount || 3;
        const fishEmojis = options.fishEmojis || ["🐠", "🐟", "🐡", "🦈", "🐋"];

        for (let index = 0; index < bubbleCount; index += 1) {
            const bubble = document.createElement("div");
            bubble.className = "bubble";
            bubble.style.left = `${Math.random() * 100}%`;
            const size = Math.random() * (options.bubbleSizeMax || 26) + (options.bubbleSizeMin || 10);
            bubble.style.width = `${size}px`;
            bubble.style.height = `${size}px`;
            bubble.style.animationDelay = `${Math.random() * 8}s`;
            bubble.style.animationDuration = `${Math.random() * 5 + 6}s`;
            container.appendChild(bubble);
        }

        for (let index = 0; index < fishCount; index += 1) {
            const fish = document.createElement("div");
            fish.className = "fish";
            fish.textContent = fishEmojis[Math.floor(Math.random() * fishEmojis.length)];
            fish.style.fontSize = `${Math.random() * (options.fishSizeRange || 20) + (options.fishSizeMin || 20)}px`;
            fish.style.top = `${Math.random() * 60 + 10}%`;
            fish.style.animationDuration = `${Math.random() * (options.fishDurationRange || 15) + (options.fishDurationMin || 15)}s`;
            fish.style.animationDelay = `${Math.random() * 20}s`;
            container.appendChild(fish);
        }
    }

    window.KuzovokShared = {
        api,
        adminUrl,
        basePath,
        bootOceanScene,
        buildPath,
        delegate,
        esc,
        formatCountdown,
        formatDate,
        highlightNode,
        homeUrl,
        profileUrl,
        renderParentPost,
        renderPostImage,
        resolveMediaUrl,
        syncImageCountdowns,
        truncateQuoted,
        updateLikeButton,
        userUrl,
        wireModalClose
    };
})(window, document);
