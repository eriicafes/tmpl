const count = document.querySelector<HTMLSpanElement>("#count")!
const counter = document.querySelector<HTMLButtonElement>("#counter")!

// increment count and save in cookies
counter.addEventListener("click", () => {
    const next = parseInt(count.innerText) + 1
    count.innerText = `${next}`
    document.cookie = `Count-State=${next};`
})