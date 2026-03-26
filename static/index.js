(function (window, document, shared) {
    const ACCESS_DENIED_MESSAGE = "извините, вы пока не кузовок";

    let currentUser = null;
    let modalReplyToPostId = null;
    let postsCache = {};

    const elements = {
        adminPanelLink: document.getElementById("adminPanelLink"),
        appContainer: document.getElementById("appContainer"),
        appContent: document.getElementById("appContent"),
        appHeader: document.getElementById("appHeader"),
        authContainer: document.getElementById("authContainer"),
        lockedMessage: document.getElementById("lockedMessage"),
        lockedState: document.getElementById("lockedState"),
        loginError: document.getElementById("loginError"),
        loginForm: document.getElementById("loginForm"),
        loginToggleLink: document.getElementById("showRegisterLink"),
        modalClearImageButton: document.getElementById("modalClearImageButton"),
        modalCloseButton: document.getElementById("modalCloseButton"),
        modalImagePreview: document.getElementById("modalImagePreview"),
        modalPostContent: document.getElementById("modalPostContent"),
        modalPostError: document.getElementById("modalPostError"),
        modalPostImageInput: document.getElementById("modalPostImageInput"),
        modalPostSuccess: document.getElementById("modalPostSuccess"),
        modalReplyBlock: document.getElementById("modalReplyBlock"),
        modalReplyContent: document.getElementById("modalReplyContent"),
        modalReplyUsername: document.getElementById("modalReplyUsername"),
        modalSelectedImageName: document.getElementById("modalSelectedImageName"),
        modalSubmitButton: document.getElementById("modalSubmitButton"),
        modalTitle: document.getElementById("modalTitle"),
        postModal: document.getElementById("postModal"),
        postsContainer: document.getElementById("postsContainer"),
        profileLink: document.getElementById("profileLink"),
        registerError: document.getElementById("registerError"),
        registerForm: document.getElementById("registerForm"),
        registerToggleLink: document.getElementById("showLoginLink"),
        siteLogo: document.getElementById("siteLogo"),
        username: document.getElementById("username")
    };

    function api(endpoint, options) {
        return shared.api(endpoint, options, {
            onResponse(response, payload) {
                if (response.status === 403 && payload && payload.message === ACCESS_DENIED_MESSAGE) {
                    if (currentUser) {
                        currentUser.is_allowed = false;
                        currentUser.access_message = payload.message;
                    }
                    applyLockedState(payload.message);
                }
            }
        });
    }

    function flashMessage(element, message) {
        if (!element) {
            return;
        }
        element.textContent = message;
        element.classList.remove("hidden");
        window.setTimeout(() => element.classList.add("hidden"), 5000);
    }

    function showLogin() {
        elements.loginForm.classList.remove("hidden");
        elements.registerForm.classList.add("hidden");
    }

    function showRegister() {
        elements.loginForm.classList.add("hidden");
        elements.registerForm.classList.remove("hidden");
    }

    function syncAdminPanelButton() {
        elements.adminPanelLink.href = shared.adminUrl();
        elements.adminPanelLink.classList.toggle("hidden", !currentUser || currentUser.role !== "admin");
    }

    function applyLockedState(message) {
        elements.appHeader.classList.add("hidden");
        elements.appContent.classList.add("hidden");
        elements.lockedState.classList.remove("hidden");
        elements.lockedMessage.textContent = message || ACCESS_DENIED_MESSAGE;
        elements.postsContainer.innerHTML = "";
        closePostModal();
        clearModalImage();
        clearModalReply();
        modalReplyToPostId = null;
    }

    function applyAllowedState() {
        elements.appHeader.classList.remove("hidden");
        elements.appContent.classList.remove("hidden");
        elements.lockedState.classList.add("hidden");
    }

    function closePostModal() {
        elements.postModal.classList.remove("active");
    }

    function clearModalReply() {
        modalReplyToPostId = null;
        elements.modalReplyBlock.classList.add("hidden");
        elements.modalTitle.textContent = "Новый пост";
    }

    function clearModalImage() {
        elements.modalPostImageInput.value = "";
        elements.modalSelectedImageName.textContent = "Без картинки";
        elements.modalClearImageButton.classList.add("hidden");
        elements.modalImagePreview.style.display = "none";
        elements.modalImagePreview.removeAttribute("src");
    }

    function handleModalImageSelection(event) {
        const file = event.target.files && event.target.files[0] ? event.target.files[0] : null;
        if (!file) {
            clearModalImage();
            return;
        }

        elements.modalSelectedImageName.textContent = file.name;
        elements.modalClearImageButton.classList.remove("hidden");

        const reader = new FileReader();
        reader.onload = function onLoad(loadEvent) {
            elements.modalImagePreview.src = loadEvent.target.result;
            elements.modalImagePreview.style.display = "block";
        };
        reader.readAsDataURL(file);
    }

    function showModalError(message) {
        flashMessage(elements.modalPostError, message);
    }

    function openPostModal(post) {
        elements.modalPostContent.value = "";
        elements.modalPostSuccess.classList.add("hidden");
        elements.modalPostError.classList.add("hidden");
        clearModalImage();

        if (post && post.id) {
            modalReplyToPostId = post.id;
            elements.modalTitle.textContent = "Ответ";
            elements.modalReplyUsername.textContent = `@${post.username}`;
            elements.modalReplyContent.textContent = shared.truncateQuoted(post.content, 20);
            elements.modalReplyBlock.classList.remove("hidden");
        } else {
            clearModalReply();
        }

        elements.postModal.classList.add("active");
        window.setTimeout(() => elements.modalPostContent.focus(), 100);
    }

    async function submitFromModal() {
        if (!currentUser || !currentUser.is_allowed) {
            applyLockedState(currentUser && currentUser.access_message);
            return;
        }

        const content = elements.modalPostContent.value.trim();
        const imageFile = elements.modalPostImageInput.files && elements.modalPostImageInput.files[0]
            ? elements.modalPostImageInput.files[0]
            : null;

        if (!content && !imageFile) {
            showModalError("Добавьте текст или картинку");
            return;
        }

        let response;
        if (imageFile) {
            const formData = new FormData();
            formData.append("content", content);
            formData.append("image", imageFile);
            if (modalReplyToPostId) {
                formData.append("parent_post_id", modalReplyToPostId);
            }
            response = await api("/posts", {
                method: "POST",
                body: formData
            });
        } else {
            const body = { content };
            if (modalReplyToPostId) {
                body.parent_post_id = modalReplyToPostId;
            }
            response = await api("/posts", {
                method: "POST",
                body: JSON.stringify(body)
            });
        }

        if (!response.success) {
            showModalError(response.message || "Не удалось создать пост");
            return;
        }

        closePostModal();
        clearModalReply();
        await loadPosts();
    }

    function scrollToPost(postId) {
        const postElement = document.getElementById(`post-${postId}`);
        if (!postElement) {
            return;
        }
        postElement.scrollIntoView({ behavior: "smooth", block: "center" });
        shared.highlightNode(postElement);
    }

    async function login() {
        const username = document.getElementById("loginUsername").value;
        const password = document.getElementById("loginPassword").value;
        const response = await api("/login", {
            method: "POST",
            body: JSON.stringify({ username, password })
        });

        if (response.success) {
            await checkAuth();
            return;
        }

        flashMessage(elements.loginError, response.message || "Не удалось войти");
    }

    async function register() {
        const username = document.getElementById("regUsername").value;
        const password = document.getElementById("regPassword").value;
        const response = await api("/register", {
            method: "POST",
            body: JSON.stringify({ username, password })
        });

        if (response.success) {
            await checkAuth();
            return;
        }

        flashMessage(elements.registerError, response.message || "Не удалось зарегистрироваться");
    }

    async function logout() {
        await api("/logout", { method: "POST" });
        currentUser = null;
        postsCache = {};
        syncAdminPanelButton();
        document.getElementById("loginUsername").value = "";
        document.getElementById("loginPassword").value = "";
        document.getElementById("regUsername").value = "";
        document.getElementById("regPassword").value = "";
        elements.authContainer.classList.remove("hidden");
        elements.appContainer.classList.add("hidden");
        applyAllowedState();
        showLogin();
    }

    function renderPost(post) {
        const likedClass = post.liked ? " liked" : "";
        const likedLabel = post.liked ? "Убрать лайк" : "Поставить лайк";
        const displayName = post.name || post.username;
        const contentHtml = post.content ? `<div class="post-content">${shared.esc(post.content)}</div>` : "";
        const imageHtml = shared.renderPostImage(post);
        const parentHtml = shared.renderParentPost(post.parent_post, { action: "scroll" });

        return `<article class="post" id="post-${post.id}">
            <div class="post-header">
                <div class="post-author">
                    <span class="post-author-avatar">${shared.esc(post.avatar || "🐠")}</span>
                    <a href="${shared.userUrl(post.username)}">${shared.esc(displayName)}</a>
                </div>
                <div class="post-date">${shared.formatDate(post.created_at)}</div>
            </div>
            ${parentHtml}
            ${contentHtml}
            ${imageHtml}
            <div class="post-footer">
                <div class="post-actions">
                    <button class="like-btn${likedClass}" type="button" aria-label="${likedLabel}" data-action="like-post" data-post-id="${post.id}">❤️ <span data-role="like-count">${post.likes}</span></button>
                    <button class="reply-btn" type="button" aria-label="Ответить" data-action="reply-post" data-post-id="${post.id}">↩</button>
                </div>
            </div>
        </article>`;
    }

    async function loadPosts() {
        if (!currentUser || !currentUser.is_allowed) {
            applyLockedState(currentUser && currentUser.access_message);
            return;
        }

        const response = await api("/feed");
        if (response.status === 403) {
            return;
        }

        if (response.success && Array.isArray(response.data) && response.data.length > 0) {
            postsCache = {};
            response.data.forEach((post) => {
                postsCache[post.id] = post;
            });
            elements.postsContainer.innerHTML = response.data.map(renderPost).join("");
            shared.syncImageCountdowns(elements.postsContainer);
            return;
        }

        postsCache = {};
        elements.postsContainer.innerHTML = '<div class="empty-state">Пока нет постов</div>';
    }

    async function toggleLike(postId, button) {
        if (!currentUser || !currentUser.is_allowed) {
            applyLockedState(currentUser && currentUser.access_message);
            return;
        }

        button.disabled = true;
        const response = await api("/like", {
            method: "POST",
            body: JSON.stringify({ post_id: Number(postId) })
        });

        if (response.success) {
            shared.updateLikeButton(button, response.data);
        }

        button.disabled = false;
    }

    function replyToPost(postId) {
        if (!currentUser || !currentUser.is_allowed) {
            applyLockedState(currentUser && currentUser.access_message);
            return;
        }

        const postData = postsCache[postId];
        if (postData) {
            openPostModal(postData);
        }
    }

    async function checkAuth() {
        const response = await api("/me");
        if (!response.success || !response.data) {
            return;
        }

        currentUser = response.data;
        elements.username.textContent = currentUser.username;
        syncAdminPanelButton();
        elements.authContainer.classList.add("hidden");
        elements.appContainer.classList.remove("hidden");

        if (currentUser.is_allowed) {
            applyAllowedState();
            await loadPosts();
            return;
        }

        applyLockedState(currentUser.access_message);
    }

    function bindEvents() {
        document.getElementById("loginButton").addEventListener("click", login);
        document.getElementById("registerButton").addEventListener("click", register);
        document.getElementById("logoutButton").addEventListener("click", logout);
        document.getElementById("openCreatePostButton").addEventListener("click", () => openPostModal());
        document.getElementById("openModalImageButton").addEventListener("click", () => elements.modalPostImageInput.click());
        document.getElementById("clearModalReplyButton").addEventListener("click", clearModalReply);
        elements.modalCloseButton.addEventListener("click", closePostModal);
        elements.modalSubmitButton.addEventListener("click", submitFromModal);
        elements.modalPostImageInput.addEventListener("change", handleModalImageSelection);
        elements.modalClearImageButton.addEventListener("click", clearModalImage);
        elements.loginToggleLink.addEventListener("click", showRegister);
        elements.registerToggleLink.addEventListener("click", showLogin);

        shared.wireModalClose(elements.postModal, closePostModal);

        shared.delegate(elements.postsContainer, "click", "[data-action='like-post']", (event, button) => {
            event.preventDefault();
            toggleLike(button.dataset.postId, button);
        });

        shared.delegate(elements.postsContainer, "click", "[data-action='reply-post']", (event, button) => {
            event.preventDefault();
            replyToPost(button.dataset.postId);
        });

        shared.delegate(elements.postsContainer, "click", "[data-parent-action='scroll']", (event, node) => {
            if (event.target.closest("a")) {
                return;
            }
            scrollToPost(node.dataset.parentId);
        });
    }

    function init() {
        shared.bootOceanScene({ bubbleCount: 15, fishCount: 5, fishSizeRange: 30, fishDurationRange: 15 });
        elements.siteLogo.src = shared.logoUrl();
        elements.profileLink.href = shared.profileUrl();
        bindEvents();
        checkAuth();
        window.setInterval(() => shared.syncImageCountdowns(elements.postsContainer), 1000);
    }

    init();
})(window, document, window.KuzovokShared);
